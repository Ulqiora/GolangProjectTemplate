package mocks

import (
	"context"
	"sync"

	kafkapkg "GolangTemplateProject/pkg/adapters/kafka"
	"github.com/IBM/sarama"
)

type Runnable struct {
	RunFunc func(ctx context.Context, group *sync.WaitGroup) error
	Calls   int
}

func (m *Runnable) Run(ctx context.Context, group *sync.WaitGroup) error {
	m.Calls++
	if m.RunFunc != nil {
		return m.RunFunc(ctx, group)
	}
	return nil
}

type TypedProducer[T any] struct {
	Topic string

	SendTypedMessageFunc    func(message kafkapkg.TypedMessage[T]) error
	SendTypedMessagesFunc   func(messages ...kafkapkg.TypedMessage[T]) error
	SendTypedMessagesTxFunc func(messages ...kafkapkg.TypedMessage[T]) error
	RunFunc                 func(ctx context.Context, group *sync.WaitGroup) error
	CloseFunc               func() error

	Messages []kafkapkg.TypedMessage[T]
	Closed   bool
}

func (m *TypedProducer[T]) TopicName() string { return m.Topic }

func (m *TypedProducer[T]) SendTypedMessage(message kafkapkg.TypedMessage[T]) error {
	m.Messages = append(m.Messages, message)
	if m.SendTypedMessageFunc != nil {
		return m.SendTypedMessageFunc(message)
	}
	return nil
}

func (m *TypedProducer[T]) SendTypedMessages(messages ...kafkapkg.TypedMessage[T]) error {
	m.Messages = append(m.Messages, messages...)
	if m.SendTypedMessagesFunc != nil {
		return m.SendTypedMessagesFunc(messages...)
	}
	return nil
}

func (m *TypedProducer[T]) SendTypedMessagesTx(messages ...kafkapkg.TypedMessage[T]) error {
	m.Messages = append(m.Messages, messages...)
	if m.SendTypedMessagesTxFunc != nil {
		return m.SendTypedMessagesTxFunc(messages...)
	}
	return nil
}

func (m *TypedProducer[T]) Run(ctx context.Context, group *sync.WaitGroup) error {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, group)
	}
	return nil
}

func (m *TypedProducer[T]) Close() error {
	m.Closed = true
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

type TransactionalProducer[T any] struct {
	TypedProducer[T]

	BeginTxFunc        func() error
	CommitTxFunc       func() error
	AbortTxFunc        func() error
	AddOffsetsToTxFunc func(offsets map[string][]*sarama.PartitionOffsetMetadata, groupID string) error
	AddMessageToTxFunc func(msg *sarama.ConsumerMessage, groupID string, metadata *string) error
}

func (m *TransactionalProducer[T]) BeginTx() error {
	if m.BeginTxFunc != nil {
		return m.BeginTxFunc()
	}
	return nil
}

func (m *TransactionalProducer[T]) CommitTx() error {
	if m.CommitTxFunc != nil {
		return m.CommitTxFunc()
	}
	return nil
}

func (m *TransactionalProducer[T]) AbortTx() error {
	if m.AbortTxFunc != nil {
		return m.AbortTxFunc()
	}
	return nil
}

func (m *TransactionalProducer[T]) AddOffsetsToTx(offsets map[string][]*sarama.PartitionOffsetMetadata, groupID string) error {
	if m.AddOffsetsToTxFunc != nil {
		return m.AddOffsetsToTxFunc(offsets, groupID)
	}
	return nil
}

func (m *TransactionalProducer[T]) AddMessageToTx(msg *sarama.ConsumerMessage, groupID string, metadata *string) error {
	if m.AddMessageToTxFunc != nil {
		return m.AddMessageToTxFunc(msg, groupID, metadata)
	}
	return nil
}

type Consumer struct {
	RunFunc func(ctx context.Context)

	Group string

	ResumeCalls int
	CloseCalls  int
	Stopped     chan struct{}
}

func (m *Consumer) Run(ctx context.Context) {
	if m.RunFunc != nil {
		m.RunFunc(ctx)
	}
}

func (m *Consumer) GroupID() string { return m.Group }

func (m *Consumer) ResumePartitions() { m.ResumeCalls++ }

func (m *Consumer) WaitStoppedSession() <-chan struct{} {
	if m.Stopped == nil {
		m.Stopped = make(chan struct{})
	}
	return m.Stopped
}

func (m *Consumer) Close() error {
	m.CloseCalls++
	return nil
}
