package producer

import (
	"errors"
	"testing"

	"GolangTemplateProject/pkg/adapters/kafka"
)

type customPayload struct {
	Value string
}

func (p customPayload) Marshal() ([]byte, error) {
	return []byte("custom:" + p.Value), nil
}

func TestDefaultSerializerUsesCustomMarshal(t *testing.T) {
	t.Parallel()

	payload, err := DefaultSerializer(customPayload{Value: "value"})
	if err != nil {
		t.Fatalf("DefaultSerializer returned error: %v", err)
	}
	if string(payload) != "custom:value" {
		t.Fatalf("unexpected payload %q", payload)
	}
}

func TestToProducerMessage(t *testing.T) {
	t.Parallel()

	msg, err := ToProducerMessage("default-topic", kafka.TypedMessage[string]{
		Key:     "key",
		Headers: map[string]string{"h": "v"},
		Value:   "payload",
	}, DefaultSerializer[string])
	if err != nil {
		t.Fatalf("ToProducerMessage returned error: %v", err)
	}
	if msg.Topic != "default-topic" {
		t.Fatalf("unexpected topic %q", msg.Topic)
	}
	if MessagePayloadSize(msg) == 0 {
		t.Fatal("expected payload size")
	}
}

func TestToProducerMessageWrapsSerializeError(t *testing.T) {
	t.Parallel()

	serializeErr := errors.New("serialize")
	_, err := ToProducerMessage("topic", kafka.TypedMessage[string]{Value: "payload"}, func(value string) ([]byte, error) {
		return nil, serializeErr
	})
	if !errors.Is(err, ErrSerializeMessagePayload) || !errors.Is(err, serializeErr) {
		t.Fatalf("expected wrapped serialize error, got %v", err)
	}
}
