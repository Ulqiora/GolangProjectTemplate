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
}

func NewTopicProducer[T any](config producer.Config, log logger.Logger, serializer producer.Serializer[T]) (*TopicProducer[T], error) {
	saramaConfig, err := producer.BuildProduceConfig(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", producer.ErrBuildSaramaConfig, err)
	}
	syncProducer, err := sarama.NewSyncProducer(config.Brokers, saramaConfig)
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
		attribute.String("topic", config.Topic),
		attribute.Int("brokers_count", len(config.Brokers)),
	)
	producerLogger.Info(producer.LogProducerConfigured)

	return &TopicProducer[T]{
		topic:      config.Topic,
		producer:   syncProducer,
		logger:     producerLogger,
		serializer: serializer,
		metrics:    producer.ResolveProducerMetrics(),
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
		return fmt.Errorf("%w: %w", producer.ErrSendMessage, err)
	}
	t.logger.Debug(producer.ErrSendMessage.Error(), attribute.String("status", "success"))
	t.metrics.ObserveOperation(producer.ProducerTypeSync, producerMessage.Topic, producer.OperationSendSingle, producer.StatusSuccess, startedAt)
	return nil
}

func (t *TopicProducer[T]) SendTypedMessages(messages ...kafka.TypedMessage[T]) error {
	startedAt := time.Now()
	producerMessages := make([]*sarama.ProducerMessage, 0, len(messages))
	for i := range messages {
		message, err := producer.ToProducerMessage(t.topic, messages[i], t.serializer)
		if err != nil {
			t.metrics.ObserveOperation(producer.ProducerTypeSync, t.topic, producer.OperationSendBatch, producer.StatusError, startedAt)
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
		return fmt.Errorf("%w: %w", producer.ErrSendMessages, err)
	}
	t.logger.Debug(
		producer.ErrSendMessages.Error(),
		attribute.String("status", "success"),
		attribute.Int("messages_count", len(producerMessages)),
	)
	t.metrics.ObserveOperation(producer.ProducerTypeSync, t.topic, producer.OperationSendBatch, producer.StatusSuccess, startedAt)
	return nil
}

func (t *TopicProducer[T]) SendTypedMessagesTx(messages ...kafka.TypedMessage[T]) error {
	return t.SendTypedMessages(messages...)
}
