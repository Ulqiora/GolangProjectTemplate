package dql

import (
	"context"
	"time"

	"GolangTemplateProject/internal/ports"
	"github.com/IBM/sarama"
)

type BaseDqlConsumerModel interface {
	ports.BaseModel
	MessageObject
}

type ExecDomainFunc[T MessageObject] func(ctx context.Context, object T, values *MapValues) error
type SaveFunction func(ctx context.Context, object *DLQMessage) error
type SaveBatchFunction func(ctx context.Context, objects []*DLQMessage) error

type Brokers []string

type MapValues map[string]string

func NewMapValues(headers []*sarama.RecordHeader) *MapValues {
	values := make(MapValues, len(headers))
	for _, header := range headers {
		values[string(header.Key)] = string(header.Value)
	}
	return &values
}

type MessageObject interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

type claimMode interface {
	HandleMessage(message *sarama.ConsumerMessage) error
	HandleFlush(reason string) error
	Timer() <-chan time.Time
	Stop()
}

type OptionFunc func(session sarama.ConsumerGroupSession) error
