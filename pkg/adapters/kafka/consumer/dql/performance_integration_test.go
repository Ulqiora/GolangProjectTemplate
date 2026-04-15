//go:build integration
// +build integration

package dql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestConsumerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("integration performance test is skipped in short mode")
	}

	env := newKafkaIntegrationEnv(t, 3*time.Minute, "dql-integration-test")
	topic, groupID := env.newTopicAndGroup("dlq-consumer-performance")
	env.createTopic(topic)

	registry := prometheus.NewRegistry()
	const messagesTotal int64 = 5000

	consumer, err := NewTopicConsumerGroupDlq[*perfMessage](
		&Config{
			Topic:   topic,
			Brokers: env.brokers,
			GroupSettings: GroupSettings{
				GroupID:                 groupID,
				OffsetInitial:           "old",
				RebalancedGroupStrategy: "round-robin",
				IsolationLevel:          "commited",
				ReturnErrors:            true,
			},
			ConsumeSettings: ConsumeProcessConfig{
				PerMessageDebugLog: false,
				LogEveryNMessages:  1000,
				BatchEnabled:       true,
				BatchSize:          100,
				BatchTimeout:       200 * time.Millisecond,
			},
		},
		env.logger,
		func(ctx context.Context, object *DLQMessage) error {
			return fmt.Errorf("unexpected DLQ write for message %s", object.ID)
		},
		func(ctx context.Context, object *perfMessage, values *MapValues) error {
			return nil
		},
		WithPrometheusRegisterer(registry),
	)
	require.NoError(t, err)

	consumerCancel := env.startConsumer(consumer)
	defer stopConsumerGracefully(t, consumer, consumerCancel)

	producer := env.newProducer()
	messages := buildPerfMessages(t, topic, messagesTotal)

	startedAt := time.Now()
	sendMessages(t, producer, messages)
	waitForCondition(t, 45*time.Second, func() bool {
		snapshot := collectPerfSnapshot(t, registry, groupID, topic)
		return snapshot.receivedTotal == float64(messagesTotal) &&
			snapshot.successTotal == float64(messagesTotal)
	}, "timeout waiting for consumer to process all performance messages")

	elapsed := time.Since(startedAt)
	throughput := float64(messagesTotal) / elapsed.Seconds()
	snapshot := collectPerfSnapshot(t, registry, groupID, topic)

	require.Equal(t, float64(messagesTotal), snapshot.receivedTotal)
	require.Equal(t, float64(messagesTotal), snapshot.successTotal)
	require.Equal(t, 0.0, snapshot.dlqTotal)
	require.Greater(t, throughput, 20.0, "throughput is too low for the integration performance baseline")

	t.Logf(
		"\nDQL consumer performance result:\n  sent=%d\n  received=%.0f\n  processed=%.0f\n  dlq_writes=%.0f\n  batches=%.0f\n  avg_batch_size=%.2f\n  elapsed=%s\n  throughput=%.2f msg/s\n  avg_batch_processing=%.6f ms/batch\n  avg_message_processing=%.6f ms/msg\n  avg_message_processing_us=%.3f us/msg",
		messagesTotal,
		snapshot.receivedTotal,
		snapshot.successTotal,
		snapshot.dlqTotal,
		snapshot.batchesProcessed,
		snapshot.avgBatchSize,
		elapsed,
		throughput,
		snapshot.avgBatchProcessingMs,
		snapshot.avgMessageProcessingMs,
		snapshot.avgMessageProcessingUs,
	)
}

func buildPerfMessages(t *testing.T, topic string, total int64) []*sarama.ProducerMessage {
	t.Helper()

	messages := make([]*sarama.ProducerMessage, 0, total)
	for i := int64(0); i < total; i++ {
		payload, err := (&perfMessage{
			ID:    fmt.Sprintf("%d", i),
			Value: fmt.Sprintf("payload-%d", i),
		}).Marshal()
		require.NoError(t, err)

		messages = append(messages, &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(fmt.Sprintf("key-%d", i)),
			Value: sarama.ByteEncoder(payload),
			Headers: []sarama.RecordHeader{
				{Key: []byte("source"), Value: []byte("integration-test")},
			},
		})
	}

	return messages
}
