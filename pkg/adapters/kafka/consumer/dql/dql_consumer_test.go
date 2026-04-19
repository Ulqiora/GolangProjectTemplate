package dql

import (
	"context"
	"testing"

	"GolangTemplateProject/pkg/logger"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewMessageObject_CreatesPointerModel(t *testing.T) {
	object, err := newMessageObject[*testMessage]()

	require.NoError(t, err)
	require.NotNil(t, object)
}

func TestNewMessageObject_ReturnsErrorForNonPointerModel(t *testing.T) {
	object, err := newMessageObject[nonPointerMessage]()

	require.Error(t, err)
	require.ErrorIs(t, err, ErrMessageObjectTypePointer)
	require.Equal(t, nonPointerMessage{}, object)
}

func TestNewTopicConsumerGroupDlq_ValidateInput(t *testing.T) {
	log, err := logger.NewLogger(logger.EnvStage, noop.NewTracerProvider().Tracer("kafka-consumer-test"))
	require.NoError(t, err)

	validConfig := &Config{
		Topic:   "topic",
		Brokers: Brokers{"localhost:9092"},
		GroupSettings: GroupSettings{
			GroupID:                 "group",
			OffsetInitial:           "new",
			RebalancedGroupStrategy: "round-robin",
			IsolationLevel:          "committed",
		},
	}

	_, err = NewTopicConsumerGroupDlq[*testMessage](nil, log, nil, func(context.Context, *testMessage, *MapValues) error { return nil })
	require.ErrorIs(t, err, ErrConfigIsNil)

	_, err = NewTopicConsumerGroupDlq[*testMessage](validConfig, log, nil, nil)
	require.ErrorIs(t, err, ErrExecDomainFuncIsNil)

	configWithDLQ := *validConfig
	configWithDLQ.ConsumeSettings.DlqSave = true
	_, err = NewTopicConsumerGroupDlq[*testMessage](&configWithDLQ, log, nil, func(context.Context, *testMessage, *MapValues) error { return nil })
	require.ErrorIs(t, err, ErrDLQSaverRequired)
}
