package producer

import (
	"errors"
	"testing"
	"time"

	"GolangTemplateProject/pkg/adapters/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestRuntimeOptionsResolveMetrics_Disabled(t *testing.T) {
	options := NewRuntimeOptions[[]byte](WithoutProducerMetrics[[]byte]())

	metrics := options.ResolveMetrics()

	require.Same(t, NoopProducerMetrics(), metrics)
	require.NotPanics(t, func() {
		metrics.ObserveOperation(ProducerTypeSync, "topic", OperationSendSingle, StatusSuccess, time.Now())
		metrics.ObservePayload(ProducerTypeSync, "topic", 128)
		metrics.ObserveAsyncEvent("topic", StatusQueued)
		metrics.ObserveMessages(ProducerTypeSync, "topic", StatusSuccess, 1)
	})
}

func TestRuntimeOptionsResolveMetrics_UsesProvidedMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	customMetrics := NewProducerMetrics(registry)

	options := NewRuntimeOptions[[]byte](WithProducerMetrics[[]byte](customMetrics))

	require.Same(t, customMetrics, options.ResolveMetrics())
}

func TestHandleError_UsesGenericHandler(t *testing.T) {
	expectedErr := errors.New("async send failed")
	expectedMessage := kafka.TypedMessage[string]{
		Topic: "topic",
		Key:   "key",
		Value: "value",
	}

	var received ErrorContext[string]
	options := NewRuntimeOptions[string](WithErrorHandler[string](func(ctx ErrorContext[string]) {
		received = ctx
	}))

	options.HandleError(OperationQueue, "topic", &expectedMessage, expectedErr)

	require.Equal(t, OperationQueue, received.Operation)
	require.Equal(t, "topic", received.Topic)
	require.Equal(t, expectedErr, received.Err)
	require.NotNil(t, received.Message)
	require.Equal(t, expectedMessage, *received.Message)
}

func TestExtractTypedMessage_ReturnsOriginalTypedMessage(t *testing.T) {
	expectedMessage := kafka.TypedMessage[string]{
		Topic: "custom.topic",
		Key:   "key-1",
		Headers: map[string]string{
			"header": "value",
		},
		Value: "payload",
	}

	producerMessage, err := ToProducerMessage("default.topic", expectedMessage, DefaultSerializer[string])
	require.NoError(t, err)

	actualMessage, ok := ExtractTypedMessage[string](producerMessage)

	require.True(t, ok)
	require.NotNil(t, actualMessage)
	require.Equal(t, expectedMessage, *actualMessage)
}
