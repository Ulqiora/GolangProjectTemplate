//go:build integration
// +build integration

package dql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"GolangTemplateProject/pkg/logger"
	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	tcKafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

type kafkaConsumerRunner interface {
	Run(context.Context)
	WaitStoppedSession() <-chan struct{}
}

type kafkaIntegrationEnv struct {
	t         *testing.T
	ctx       context.Context
	cancel    context.CancelFunc
	container *tcKafka.KafkaContainer
	brokers   []string
	logger    logger.Logger
}

type perfSnapshot struct {
	receivedTotal          float64
	successTotal           float64
	decodeFailedTotal      float64
	processingFailedTotal  float64
	dlqTotal               float64
	batchesProcessed       float64
	avgBatchSize           float64
	avgBatchProcessingMs   float64
	avgMessageProcessingMs float64
	avgMessageProcessingUs float64
}

func newKafkaIntegrationEnv(t *testing.T, timeout time.Duration, tracerName string) *kafkaIntegrationEnv {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	container, err := runKafkaContainerForTest(ctx)
	if err != nil {
		cancel()
		t.Skipf("skipping integration test: Docker/testcontainers unavailable: %v", err)
	}

	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
		cancel()
	})

	brokers, err := container.Brokers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, brokers)

	testLogger, err := logger.NewLogger(logger.EnvStage, noop.NewTracerProvider().Tracer(tracerName))
	require.NoError(t, err)

	return &kafkaIntegrationEnv{
		t:         t,
		ctx:       ctx,
		cancel:    cancel,
		container: container,
		brokers:   brokers,
		logger:    testLogger,
	}
}

func (e *kafkaIntegrationEnv) newTopicAndGroup(prefix string) (string, string) {
	e.t.Helper()

	return prefix + "-" + uuid.NewString(), prefix + "-group-" + uuid.NewString()
}

func (e *kafkaIntegrationEnv) createTopic(topic string) {
	e.t.Helper()

	admin, err := sarama.NewClusterAdmin(e.brokers, sarama.NewConfig())
	require.NoError(e.t, err)
	defer func() {
		require.NoError(e.t, admin.Close())
	}()

	err = admin.CreateTopic(topic, &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}, false)
	require.NoError(e.t, err)
}

func (e *kafkaIntegrationEnv) newProducer() sarama.SyncProducer {
	e.t.Helper()

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	producer, err := sarama.NewSyncProducer(e.brokers, config)
	require.NoError(e.t, err)
	e.t.Cleanup(func() {
		require.NoError(e.t, producer.Close())
	})

	return producer
}

func (e *kafkaIntegrationEnv) startConsumer(consumer kafkaConsumerRunner) context.CancelFunc {
	e.t.Helper()

	consumerCtx, consumerCancel := context.WithCancel(e.ctx)
	consumer.Run(consumerCtx)

	time.Sleep(5 * time.Second)

	return consumerCancel
}

func stopConsumerGracefully(t *testing.T, consumer kafkaConsumerRunner, cancel context.CancelFunc) {
	t.Helper()

	cancel()
	waitUntilDone(t, consumer.WaitStoppedSession(), 15*time.Second, "consumer did not stop gracefully")
}

func sendMessages(t *testing.T, producer sarama.SyncProducer, messages []*sarama.ProducerMessage) {
	t.Helper()
	require.NoError(t, producer.SendMessages(messages))
}

func waitUntilDone(t *testing.T, done <-chan struct{}, timeout time.Duration, failureMessage string, args ...any) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatalf(failureMessage, args...)
	}
}

func collectPerfSnapshot(t *testing.T, registry *prometheus.Registry, groupID string, topic string) perfSnapshot {
	t.Helper()

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	avgBatchProcessingSeconds := findHistogramAverage(metricFamilies, "dlq_consumer_batch_duration_seconds", map[string]string{
		"group_id": groupID,
		"topic":    topic,
		"reason":   "batch_size",
	})
	if avgBatchProcessingSeconds == 0 {
		avgBatchProcessingSeconds = findHistogramAverage(metricFamilies, "dlq_consumer_batch_duration_seconds", map[string]string{
			"group_id": groupID,
			"topic":    topic,
			"reason":   "batch_timeout",
		})
	}

	avgProcessingSeconds := findHistogramAverage(metricFamilies, "dlq_consumer_processing_duration_seconds", map[string]string{
		"group_id": groupID,
		"topic":    topic,
		"status":   "success",
	})

	return perfSnapshot{
		receivedTotal: requireMetricCounterValue(t, metricFamilies, "dlq_consumer_messages_received_total", map[string]string{
			"group_id": groupID,
			"topic":    topic,
		}, -1),
		successTotal: requireMetricCounterValue(t, metricFamilies, "dlq_consumer_messages_processed_total", map[string]string{
			"group_id": groupID,
			"topic":    topic,
			"status":   "success",
		}, -1),
		decodeFailedTotal: findMetricCounterValue(metricFamilies, "dlq_consumer_messages_processed_total", map[string]string{
			"group_id": groupID,
			"topic":    topic,
			"status":   "decode_failed",
		}),
		processingFailedTotal: findMetricCounterValue(metricFamilies, "dlq_consumer_messages_processed_total", map[string]string{
			"group_id": groupID,
			"topic":    topic,
			"status":   "processing_failed",
		}),
		dlqTotal: sumMetricCounterValues(metricFamilies, "dlq_consumer_dlq_writes_total", map[string]string{
			"group_id": groupID,
			"topic":    topic,
			"status":   "success",
		}),
		batchesProcessed: sumMetricCounterValues(metricFamilies, "dlq_consumer_batches_processed_total", map[string]string{
			"group_id": groupID,
			"topic":    topic,
		}),
		avgBatchSize: findHistogramAverage(metricFamilies, "dlq_consumer_batch_size", map[string]string{
			"group_id": groupID,
			"topic":    topic,
		}),
		avgBatchProcessingMs:   avgBatchProcessingSeconds * 1000,
		avgMessageProcessingMs: avgProcessingSeconds * 1000,
		avgMessageProcessingUs: avgProcessingSeconds * 1_000_000,
	}
}

func requireMetricCounterValue(t *testing.T, metricFamilies []*dto.MetricFamily, name string, labels map[string]string, expected float64) float64 {
	t.Helper()

	for _, family := range metricFamilies {
		if family.GetName() != name {
			continue
		}

		for _, metric := range family.Metric {
			if labelsMatch(metric, labels) {
				require.NotNil(t, metric.Counter)
				value := metric.Counter.GetValue()
				if expected >= 0 {
					require.Equal(t, expected, value)
				}
				return value
			}
		}
	}

	t.Fatalf("metric %s with labels %v not found", name, labels)
	return 0
}

func findMetricCounterValue(metricFamilies []*dto.MetricFamily, name string, labels map[string]string) float64 {
	for _, family := range metricFamilies {
		if family.GetName() != name {
			continue
		}

		for _, metric := range family.Metric {
			if labelsMatch(metric, labels) && metric.Counter != nil {
				return metric.Counter.GetValue()
			}
		}
	}

	return 0
}

func sumMetricCounterValues(metricFamilies []*dto.MetricFamily, name string, labels map[string]string) float64 {
	total := 0.0
	for _, family := range metricFamilies {
		if family.GetName() != name {
			continue
		}

		for _, metric := range family.Metric {
			if partialLabelsMatch(metric, labels) && metric.Counter != nil {
				total += metric.Counter.GetValue()
			}
		}
	}

	return total
}

func findHistogramAverage(metricFamilies []*dto.MetricFamily, name string, labels map[string]string) float64 {
	for _, family := range metricFamilies {
		if family.GetName() != name {
			continue
		}

		for _, metric := range family.Metric {
			if !labelsMatch(metric, labels) || metric.Histogram == nil {
				continue
			}

			sampleCount := metric.Histogram.GetSampleCount()
			if sampleCount == 0 {
				return 0
			}

			return metric.Histogram.GetSampleSum() / float64(sampleCount)
		}
	}

	return 0
}

func runKafkaContainerForTest(ctx context.Context) (container *tcKafka.KafkaContainer, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("testcontainers panic: %v", recovered)
		}
	}()

	return tcKafka.Run(
		ctx,
		"confluentinc/confluent-local:7.5.0",
		tcKafka.WithClusterID("dlq-consumer-performance"),
	)
}

func labelsMatch(metric *dto.Metric, expected map[string]string) bool {
	if len(metric.Label) != len(expected) {
		return false
	}

	for _, label := range metric.Label {
		value, ok := expected[label.GetName()]
		if !ok || value != label.GetValue() {
			return false
		}
	}

	return true
}

func partialLabelsMatch(metric *dto.Metric, expected map[string]string) bool {
	actual := make(map[string]string, len(metric.Label))
	for _, label := range metric.Label {
		actual[label.GetName()] = label.GetValue()
	}

	for key, value := range expected {
		if actual[key] != value {
			return false
		}
	}

	return true
}
