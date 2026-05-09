package async_producer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/adapters/kafka/producer"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	"github.com/IBM/sarama"
)

type TopicProducer[T any] struct {
	topic      string
	producer   sarama.AsyncProducer
	serializer producer.Serializer[T]
	logger     logger.Logger
	metrics    *producer.Metrics
	options    producer.RuntimeOptions[T]
}

func NewTopicProducer[T any](config producer.Config, log logger.Logger, serializer producer.Serializer[T], opts ...producer.ProducerOption[T]) (*TopicProducer[T], error) {
	saramaConfig, err := producer.BuildProduceConfig(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", producer.ErrBuildSaramaConfig, err)
	}
	asyncProducer, err := sarama.NewAsyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", producer.ErrCreateAsyncProducer, err)
	}
	if serializer == nil {
		serializer = producer.DefaultSerializer[T]
	}
	baseLogger := log
	if baseLogger == nil {
		baseLogger = logger.DefaultLogger()
	}
	if baseLogger == nil {
		return nil, producer.ErrLoggerIsNil
	}

	producerLogger := baseLogger.With(
		attribute.String("topic", config.Topic),
		attribute.Int("brokers_count", len(config.Brokers)),
	)
	producerLogger.Info(producer.LogProducerConfigured)

	resolvedOptions := producer.NewRuntimeOptions(opts...)

	return &TopicProducer[T]{
		topic:      config.Topic,
		producer:   asyncProducer,
		serializer: serializer,
		logger:     producerLogger,
		metrics:    resolvedOptions.ResolveMetrics(),
		options:    resolvedOptions,
	}, nil
}

func (t *TopicProducer[T]) TopicName() string {
	return t.topic
}

func (t *TopicProducer[T]) Run(ctx context.Context, group *sync.WaitGroup) error {
	if group == nil {
		return producer.ErrWaitGroupIsNil
	}

	group.Add(1)
	t.logger.Info(producer.LogAsyncLoopStarted)

	go func() {
		defer group.Done()

		errorsCh := t.producer.Errors()
		successesCh := t.producer.Successes()

		for {
			select {
			case err, ok := <-errorsCh:
				if !ok {
					errorsCh = nil
					t.logger.Info(producer.LogAsyncErrorsChannelClosed)
					if successesCh == nil {
						t.logger.Info(producer.LogAsyncLoopStopped)
						return
					}
					continue
				}
				if err == nil {
					continue
				}
				topic := t.topic
				if err.Msg != nil && err.Msg.Topic != "" {
					topic = err.Msg.Topic
				}
				t.metrics.ObserveAsyncEvent(topic, producer.StatusError)
				t.metrics.ObserveOperation(producer.ProducerTypeAsync, topic, producer.OperationQueue, producer.StatusError, time.Now())
				t.metrics.ObserveMessages(producer.ProducerTypeAsync, topic, producer.StatusError, 1)
				typedMessage, _ := producer.ExtractTypedMessage[T](err.Msg)
				t.options.HandleError(producer.OperationQueue, topic, typedMessage, err.Err)
				t.logger.Error(
					producer.LogAsyncProducerReturnedError,
					attribute.String("error", err.Err.Error()),
				)
			case msg, ok := <-successesCh:
				if !ok {
					successesCh = nil
					t.logger.Info(producer.LogAsyncSuccessesChannelClosed)
					if errorsCh == nil {
						t.logger.Info(producer.LogAsyncLoopStopped)
						return
					}
					continue
				}
				if msg == nil {
					continue
				}
				t.metrics.ObserveAsyncEvent(msg.Topic, producer.StatusAcked)
				t.metrics.ObserveOperation(producer.ProducerTypeAsync, msg.Topic, producer.OperationQueue, producer.StatusAcked, time.Now())
				t.metrics.ObserveMessages(producer.ProducerTypeAsync, msg.Topic, producer.StatusAcked, 1)
				t.logger.Debug(
					producer.LogAsyncProducerDeliveredMessage,
					attribute.String("topic", msg.Topic),
					attribute.Int("partition", int(msg.Partition)),
					attribute.Int64("offset", msg.Offset),
				)
			case <-ctx.Done():
				t.logger.Info(producer.LogAsyncLoopStopped)
				return
			}
		}
	}()
	return nil
}

func (t *TopicProducer[T]) SendTypedMessage(message kafka.TypedMessage[T]) error {
	startedAt := time.Now()
	producerMessage, err := producer.ToProducerMessage(t.topic, message, t.serializer)
	if err != nil {
		t.metrics.ObserveOperation(producer.ProducerTypeAsync, t.topic, producer.OperationQueue, producer.StatusError, startedAt)
		t.options.HandleError(producer.OperationQueue, t.topic, &message, err)
		return err
	}
	t.producer.Input() <- producerMessage
	t.metrics.ObservePayload(producer.ProducerTypeAsync, producerMessage.Topic, producer.MessagePayloadSize(producerMessage))
	t.metrics.ObserveOperation(producer.ProducerTypeAsync, producerMessage.Topic, producer.OperationQueue, producer.StatusQueued, startedAt)
	t.metrics.ObserveAsyncEvent(producerMessage.Topic, producer.StatusQueued)
	t.metrics.ObserveMessages(producer.ProducerTypeAsync, producerMessage.Topic, producer.StatusQueued, 1)
	t.logger.Debug(producer.LogProducerMessageQueued, attribute.String("status", "queued"))
	return nil
}

func (t *TopicProducer[T]) SendTypedMessages(messages ...kafka.TypedMessage[T]) error {
	startedAt := time.Now()
	topic := t.topic
	for i := range messages {
		producerMessage, err := producer.ToProducerMessage(t.topic, messages[i], t.serializer)
		if err != nil {
			t.metrics.ObserveOperation(producer.ProducerTypeAsync, topic, producer.OperationSendBatch, producer.StatusError, startedAt)
			t.options.HandleError(producer.OperationSendBatch, topic, &messages[i], err)
			return err
		}
		t.producer.Input() <- producerMessage
		topic = producerMessage.Topic
		t.metrics.ObservePayload(producer.ProducerTypeAsync, producerMessage.Topic, producer.MessagePayloadSize(producerMessage))
		t.metrics.ObserveAsyncEvent(producerMessage.Topic, producer.StatusQueued)
		t.metrics.ObserveMessages(producer.ProducerTypeAsync, producerMessage.Topic, producer.StatusQueued, 1)
	}
	t.metrics.ObserveOperation(producer.ProducerTypeAsync, topic, producer.OperationSendBatch, producer.StatusQueued, startedAt)
	t.logger.Debug(
		producer.LogProducerMessagesSent,
		attribute.String("status", "queued"),
		attribute.Int("messages_count", len(messages)),
	)
	return nil
}

func (t *TopicProducer[T]) SendTypedMessagesTx(messages ...kafka.TypedMessage[T]) error {
	return t.SendTypedMessages(messages...)
}

func (t *TopicProducer[T]) Close() error {
	if err := t.producer.Close(); err != nil {
		return err
	}
	t.logger.Info(producer.LogProducerClosed)
	return nil
}
