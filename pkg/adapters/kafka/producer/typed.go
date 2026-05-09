package producer

import (
	"encoding/json"
	"fmt"

	"GolangTemplateProject/pkg/adapters/kafka"
	"github.com/IBM/sarama"
)

type Marshaler interface {
	Marshal() ([]byte, error)
}

type Serializer[T any] func(value T) ([]byte, error)

type producerMessageMetadata[T any] struct {
	typedMessage kafka.TypedMessage[T]
}

func DefaultSerializer[T any](value T) ([]byte, error) {
	if marshalValue, ok := any(value).(Marshaler); ok {
		return marshalValue.Marshal()
	}
	return json.Marshal(value)
}

func ToProducerMessage[T any](defaultTopic string, message kafka.TypedMessage[T], serializer Serializer[T]) (*sarama.ProducerMessage, error) {
	payload, err := serializer(message.Value)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSerializeMessagePayload, err)
	}

	producerMessage := &sarama.ProducerMessage{
		Topic: resolveTopic(defaultTopic, message.Topic),
		Value: sarama.ByteEncoder(payload),
		Metadata: producerMessageMetadata[T]{
			typedMessage: kafka.TypedMessage[T]{
				Topic:   resolveTopic(defaultTopic, message.Topic),
				Key:     message.Key,
				Headers: cloneHeaders(message.Headers),
				Value:   message.Value,
			},
		},
	}
	if message.Key != "" {
		producerMessage.Key = sarama.StringEncoder(message.Key)
	}
	if len(message.Headers) > 0 {
		producerMessage.Headers = make([]sarama.RecordHeader, 0, len(message.Headers))
		for key, value := range message.Headers {
			producerMessage.Headers = append(producerMessage.Headers, sarama.RecordHeader{
				Key:   []byte(key),
				Value: []byte(value),
			})
		}
	}
	return producerMessage, nil
}

func ExtractTypedMessage[T any](message *sarama.ProducerMessage) (*kafka.TypedMessage[T], bool) {
	if message == nil || message.Metadata == nil {
		return nil, false
	}

	if metadata, ok := message.Metadata.(producerMessageMetadata[T]); ok {
		typedMessage := metadata.typedMessage
		return &typedMessage, true
	}
	if metadata, ok := message.Metadata.(*producerMessageMetadata[T]); ok && metadata != nil {
		typedMessage := metadata.typedMessage
		return &typedMessage, true
	}

	return nil, false
}

func resolveTopic(defaultTopic string, messageTopic string) string {
	if messageTopic != "" {
		return messageTopic
	}
	return defaultTopic
}

func MessagePayloadSize(message *sarama.ProducerMessage) int {
	if message == nil || message.Value == nil {
		return 0
	}

	payload, err := message.Value.Encode()
	if err != nil {
		return 0
	}

	return len(payload)
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}

	return cloned
}
