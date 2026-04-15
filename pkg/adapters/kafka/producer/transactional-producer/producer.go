package transactional_producer

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
	if config.ProduceSettings.TransactionalID == "" {
		return nil, producer.ErrTransactionalIDRequired
	}

	cfg := config
	cfg.ProduceSettings.Idempotency = true
	cfg.ProduceSettings.RequiredAcks = int16(sarama.WaitForAll)
	cfg.ProduceSettings.SaveReturningStatus.Errors = true
	cfg.ProduceSettings.SaveReturningStatus.Succeeded = true

	saramaConfig, err := producer.BuildProduceConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", producer.ErrBuildSaramaConfig, err)
	}

	syncProducer, err := sarama.NewSyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", producer.ErrCreateTxProducer, err)
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
		attribute.String("transactional_id", cfg.ProduceSettings.TransactionalID),
		attribute.Int("brokers_count", len(cfg.Brokers)),
	)
	producerLogger.Info(producer.LogTransactionalProducerConfigured)

	return &TopicProducer[T]{
		topic:      cfg.Topic,
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

func (t *TopicProducer[T]) sendProducerMessage(message *sarama.ProducerMessage) error {
	startedAt := time.Now()
	t.metrics.ObservePayload(producer.ProducerTypeTransactional, message.Topic, producer.MessagePayloadSize(message))
	_, _, err := t.producer.SendMessage(message)
	if err != nil {
		t.logger.Error(
			producer.ErrSendMessage.Error(),
			attribute.String("error", err.Error()),
		)
		t.metrics.ObserveOperation(producer.ProducerTypeTransactional, message.Topic, producer.OperationSendSingle, producer.StatusError, startedAt)
		return fmt.Errorf("%w: %w", producer.ErrSendMessage, err)
	}
	t.logger.Debug(producer.ErrSendMessage.Error(), attribute.String("status", "success"))
	t.metrics.ObserveOperation(producer.ProducerTypeTransactional, message.Topic, producer.OperationSendSingle, producer.StatusSuccess, startedAt)
	return nil
}

func (t *TopicProducer[T]) sendProducerMessages(messages ...*sarama.ProducerMessage) error {
	startedAt := time.Now()
	if err := t.producer.SendMessages(messages); err != nil {
		t.logger.Error(
			producer.ErrSendMessages.Error(),
			attribute.String("error", err.Error()),
			attribute.Int("messages_count", len(messages)),
		)
		t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationSendBatch, producer.StatusError, startedAt)
		return fmt.Errorf("%w: %w", producer.ErrSendMessages, err)
	}
	for i := range messages {
		t.metrics.ObservePayload(producer.ProducerTypeTransactional, messages[i].Topic, producer.MessagePayloadSize(messages[i]))
	}
	t.logger.Debug(
		producer.ErrSendMessages.Error(),
		attribute.String("status", "success"),
		attribute.Int("messages_count", len(messages)),
	)
	t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationSendBatch, producer.StatusSuccess, startedAt)
	return nil
}

func (t *TopicProducer[T]) SendTypedMessage(message kafka.TypedMessage[T]) error {
	producerMessage, err := producer.ToProducerMessage(t.topic, message, t.serializer)
	if err != nil {
		return err
	}
	return t.sendProducerMessage(producerMessage)
}

func (t *TopicProducer[T]) SendTypedMessages(messages ...kafka.TypedMessage[T]) error {
	producerMessages := make([]*sarama.ProducerMessage, 0, len(messages))
	for i := range messages {
		producerMessage, err := producer.ToProducerMessage(t.topic, messages[i], t.serializer)
		if err != nil {
			return err
		}
		producerMessages = append(producerMessages, producerMessage)
	}
	return t.sendProducerMessages(producerMessages...)
}

func (t *TopicProducer[T]) SendTypedMessagesTx(messages ...kafka.TypedMessage[T]) error {
	if err := t.BeginTx(); err != nil {
		return err
	}

	if err := t.SendTypedMessages(messages...); err != nil {
		abortErr := t.AbortTx()
		if abortErr != nil {
			return fmt.Errorf("%w: %w, %w: %w", producer.ErrSendMessages, err, producer.ErrAbortTransaction, abortErr)
		}
		t.logger.Warn(producer.LogTxSendFailedAbortSucceeded, attribute.String("error", err.Error()))
		return fmt.Errorf("%w: %w", producer.ErrSendMessages, err)
	}

	if err := t.CommitTx(); err != nil {
		if t.producer.TxnStatus()&sarama.ProducerTxnFlagAbortableError != 0 {
			abortErr := t.AbortTx()
			if abortErr != nil {
				return fmt.Errorf("%w: %w, %w: %w", producer.ErrCommitTransaction, err, producer.ErrAbortTransaction, abortErr)
			}
			t.logger.Warn(producer.LogTxCommitFailedAbortSucceeded, attribute.String("error", err.Error()))
		}
		return fmt.Errorf("%w: %w", producer.ErrCommitTransaction, err)
	}
	return nil
}

func (t *TopicProducer[T]) BeginTx() error {
	startedAt := time.Now()
	if err := t.producer.BeginTxn(); err != nil {
		t.logger.Error(
			producer.ErrBeginTransaction.Error(),
			attribute.String("error", err.Error()),
		)
		t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationBeginTx, producer.StatusError, startedAt)
		return fmt.Errorf("%w: %w", producer.ErrBeginTransaction, err)
	}
	t.logger.Debug(producer.ErrBeginTransaction.Error(), attribute.String("status", "success"))
	t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationBeginTx, producer.StatusSuccess, startedAt)
	return nil
}

func (t *TopicProducer[T]) CommitTx() error {
	startedAt := time.Now()
	if err := t.producer.CommitTxn(); err != nil {
		t.logger.Error(
			producer.ErrCommitTransaction.Error(),
			attribute.String("error", err.Error()),
		)
		t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationCommitTx, producer.StatusError, startedAt)
		return fmt.Errorf("%w: %w", producer.ErrCommitTransaction, err)
	}
	t.logger.Debug(producer.ErrCommitTransaction.Error(), attribute.String("status", "success"))
	t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationCommitTx, producer.StatusSuccess, startedAt)
	return nil
}

func (t *TopicProducer[T]) AbortTx() error {
	startedAt := time.Now()
	if err := t.producer.AbortTxn(); err != nil {
		t.logger.Error(
			producer.ErrAbortTransaction.Error(),
			attribute.String("error", err.Error()),
		)
		t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationAbortTx, producer.StatusError, startedAt)
		return fmt.Errorf("%w: %w", producer.ErrAbortTransaction, err)
	}
	t.logger.Warn(producer.ErrAbortTransaction.Error(), attribute.String("status", "success"))
	t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationAbortTx, producer.StatusSuccess, startedAt)
	return nil
}

func (t *TopicProducer[T]) AddOffsetsToTx(offsets map[string][]*sarama.PartitionOffsetMetadata, groupID string) error {
	startedAt := time.Now()
	if err := t.producer.AddOffsetsToTxn(offsets, groupID); err != nil {
		t.logger.Error(
			producer.ErrAddOffsetsToTransaction.Error(),
			attribute.String("group_id", groupID),
			attribute.String("error", err.Error()),
		)
		t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationAddOffsets, producer.StatusError, startedAt)
		return fmt.Errorf("%w: %w", producer.ErrAddOffsetsToTransaction, err)
	}
	t.logger.Debug(
		producer.ErrAddOffsetsToTransaction.Error(),
		attribute.String("group_id", groupID),
		attribute.String("status", "success"),
	)
	t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationAddOffsets, producer.StatusSuccess, startedAt)
	return nil
}

func (t *TopicProducer[T]) AddMessageToTx(msg *sarama.ConsumerMessage, groupID string, metadata *string) error {
	startedAt := time.Now()
	if err := t.producer.AddMessageToTxn(msg, groupID, metadata); err != nil {
		t.logger.Error(
			producer.ErrAddMessageToTransaction.Error(),
			attribute.String("group_id", groupID),
			attribute.String("error", err.Error()),
		)
		t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationAddMessage, producer.StatusError, startedAt)
		return fmt.Errorf("%w: %w", producer.ErrAddMessageToTransaction, err)
	}
	t.logger.Debug(
		producer.ErrAddMessageToTransaction.Error(),
		attribute.String("group_id", groupID),
		attribute.String("status", "success"),
	)
	t.metrics.ObserveOperation(producer.ProducerTypeTransactional, t.topic, producer.OperationAddMessage, producer.StatusSuccess, startedAt)
	return nil
}

func (t *TopicProducer[T]) Close() error {
	return t.producer.Close()
}
