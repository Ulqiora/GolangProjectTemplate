package sync_producer

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
	producer   sarama.SyncProducer
	logger     logger.Logger
	serializer producer.Serializer[T]
	metrics    *producer.Metrics
	options    producer.RuntimeOptions[T]
}

func NewTopicProducer[T any](config producer.Config, log logger.Logger, serializer producer.Serializer[T], opts ...producer.ProducerOption[T]) (*TopicProducer[T], error) {
	cfg := config
	cfg.ProduceSettings.SaveReturningStatus.Errors = true
	cfg.ProduceSettings.SaveReturningStatus.Succeeded = true

	saramaConfig, err := producer.BuildProduceConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", producer.ErrBuildSaramaConfig, err)
	}
	syncProducer, err := sarama.NewSyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", producer.ErrCreateSyncProducer, err)
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
		attribute.String("topic", cfg.Topic),
		attribute.Int("brokers_count", len(cfg.Brokers)),
	)
	producerLogger.Info(producer.LogProducerConfigured)

	resolvedOptions := producer.NewRuntimeOptions(opts...)

	return &TopicProducer[T]{
		topic:      cfg.Topic,
		producer:   syncProducer,
		logger:     producerLogger,
		serializer: serializer,
		metrics:    resolvedOptions.ResolveMetrics(),
		options:    resolvedOptions,
	}, nil
}

func (t *TopicProducer[T]) TopicName() string {
	return t.topic
}

func (t *TopicProducer[T]) Run(_ context.Context, _ *sync.WaitGroup) error {
	return nil
}

func (t *TopicProducer[T]) SendTypedMessage(message kafka.TypedMessage[T]) error {
	startedAt := time.Now()
	producerMessage, err := producer.ToProducerMessage(t.topic, message, t.serializer)
	if err != nil {
		t.metrics.ObserveOperation(producer.ProducerTypeSync, t.topic, producer.OperationSendSingle, producer.StatusError, startedAt)
		t.options.HandleError(producer.OperationSendSingle, t.topic, &message, err)
		return err
	}
	t.metrics.ObservePayload(producer.ProducerTypeSync, producerMessage.Topic, producer.MessagePayloadSize(producerMessage))
	_, _, err = t.producer.SendMessage(producerMessage)
	if err != nil {
		t.logger.Error(
			producer.ErrSendMessage.Error(),
			attribute.String("error", err.Error()),
		)
		t.metrics.ObserveOperation(producer.ProducerTypeSync, producerMessage.Topic, producer.OperationSendSingle, producer.StatusError, startedAt)
		t.options.HandleError(producer.OperationSendSingle, producerMessage.Topic, &message, err)
		return fmt.Errorf("%w: %w", producer.ErrSendMessage, err)
	}
	t.logger.Debug(producer.LogProducerMessageSent, attribute.String("status", "success"))
	t.metrics.ObserveOperation(producer.ProducerTypeSync, producerMessage.Topic, producer.OperationSendSingle, producer.StatusSuccess, startedAt)
	t.metrics.ObserveMessages(producer.ProducerTypeSync, producerMessage.Topic, producer.StatusSuccess, 1)
	return nil
}

func (t *TopicProducer[T]) SendTypedMessages(messages ...kafka.TypedMessage[T]) error {
	startedAt := time.Now()
	producerMessages := make([]*sarama.ProducerMessage, 0, len(messages))
	for i := range messages {
		message, err := producer.ToProducerMessage(t.topic, messages[i], t.serializer)
		if err != nil {
			t.metrics.ObserveOperation(producer.ProducerTypeSync, t.topic, producer.OperationSendBatch, producer.StatusError, startedAt)
			t.options.HandleError(producer.OperationSendBatch, t.topic, &messages[i], err)
			return err
		}
		producerMessages = append(producerMessages, message)
		t.metrics.ObservePayload(producer.ProducerTypeSync, message.Topic, producer.MessagePayloadSize(message))
	}

	if err := t.producer.SendMessages(producerMessages); err != nil {
		t.logger.Error(
			producer.ErrSendMessages.Error(),
			attribute.String("error", err.Error()),
			attribute.Int("messages_count", len(producerMessages)),
		)
		t.metrics.ObserveOperation(producer.ProducerTypeSync, t.topic, producer.OperationSendBatch, producer.StatusError, startedAt)
		for i := range messages {
			t.options.HandleError(producer.OperationSendBatch, producerMessages[i].Topic, &messages[i], err)
		}
		return fmt.Errorf("%w: %w", producer.ErrSendMessages, err)
	}
	t.logger.Debug(
		producer.LogProducerMessagesSent,
		attribute.String("status", "success"),
		attribute.Int("messages_count", len(producerMessages)),
	)
	t.metrics.ObserveOperation(producer.ProducerTypeSync, t.topic, producer.OperationSendBatch, producer.StatusSuccess, startedAt)
	t.metrics.ObserveMessages(producer.ProducerTypeSync, t.topic, producer.StatusSuccess, len(producerMessages))
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
