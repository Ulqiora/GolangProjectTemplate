package kafka

import (
	"context"

	"github.com/IBM/sarama"
)

type TypedMessage[T any] struct {
	Topic   string
	Key     string
	Headers map[string]string
	Value   T
}

type TypedProducer[T any] interface {
	TopicName() string
	SendTypedMessage(message TypedMessage[T]) error
	SendTypedMessages(messages ...TypedMessage[T]) error
	SendTypedMessagesTx(messages ...TypedMessage[T]) error
	Runnable
}

// ClosableTypedProducer extends TypedProducer with explicit resource cleanup.
// It is kept as a separate interface to preserve backward compatibility.
type ClosableTypedProducer[T any] interface {
	TypedProducer[T]
	Close() error
}

type TransactionalProducer[T any] interface {
	TypedProducer[T]
	BeginTx() error
	CommitTx() error
	AbortTx() error
	AddOffsetsToTx(offsets map[string][]*sarama.PartitionOffsetMetadata, groupID string) error
	AddMessageToTx(msg *sarama.ConsumerMessage, groupID string, metadata *string) error
}

type Consumer interface {
	Run(ctx context.Context)
	GroupID() string
	ResumePartitions()
	WaitStoppedSession() <-chan struct{}
}

// ClosableConsumer extends Consumer with explicit resource cleanup.
// It is kept as a separate interface to preserve backward compatibility.
type ClosableConsumer interface {
	Consumer
	Close() error
}
