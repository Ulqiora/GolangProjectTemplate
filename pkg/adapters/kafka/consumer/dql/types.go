package dql

import (
	"context"

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

type OptionFunc func(session sarama.ConsumerGroupSession) error
