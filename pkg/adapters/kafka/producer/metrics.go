package producer

import (
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultMetricsNamespace = "kafka_producer"

	ProducerTypeSync          = "sync"
	ProducerTypeAsync         = "async"
	ProducerTypeTransactional = "transactional"

	OperationSendSingle = "send_single"
	OperationSendBatch  = "send_batch"
	OperationQueue      = "queue"
	OperationBeginTx    = "begin_tx"
	OperationCommitTx   = "commit_tx"
	OperationAbortTx    = "abort_tx"
	OperationAddOffsets = "add_offsets"
	OperationAddMessage = "add_message"

	StatusSuccess = "success"
	StatusError   = "error"
	StatusQueued  = "queued"
	StatusAcked   = "acked"
)

type Metrics struct {
	operationsTotal   *prometheus.CounterVec
	operationDuration *prometheus.HistogramVec
	payloadBytes      *prometheus.HistogramVec
	asyncEvents       *prometheus.CounterVec
}

var (
	defaultMetricsOnce sync.Once
	defaultMetrics     *Metrics
)

func ResolveProducerMetrics() *Metrics {
	defaultMetricsOnce.Do(func() {
		defaultMetrics = newProducerMetrics(prometheus.DefaultRegisterer)
	})

	return defaultMetrics
}

func newProducerMetrics(registerer prometheus.Registerer) *Metrics {
	return &Metrics{
		operationsTotal: mustRegisterCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "operations_total",
				Help:      "Total number of producer operations by type, topic, operation and status.",
			},
			[]string{"producer_type", "topic", "operation", "status"},
		)),
		operationDuration: mustRegisterHistogramVec(registerer, prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "operation_duration_seconds",
				Help:      "Duration of producer operations in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"producer_type", "topic", "operation", "status"},
		)),
		payloadBytes: mustRegisterHistogramVec(registerer, prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "payload_bytes",
				Help:      "Payload size in bytes for messages sent by producers.",
				Buckets:   []float64{32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576},
			},
			[]string{"producer_type", "topic"},
		)),
		asyncEvents: mustRegisterCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "async_events_total",
				Help:      "Total number of async producer queue, ack and error events.",
			},
			[]string{"topic", "event"},
		)),
	}
}

func (m *Metrics) ObserveOperation(producerType, topic, operation, status string, startedAt time.Time) {
	m.operationsTotal.WithLabelValues(producerType, topic, operation, status).Inc()
	m.operationDuration.WithLabelValues(producerType, topic, operation, status).Observe(time.Since(startedAt).Seconds())
}

func (m *Metrics) ObservePayload(producerType, topic string, bytes int) {
	if bytes <= 0 {
		return
	}
	m.payloadBytes.WithLabelValues(producerType, topic).Observe(float64(bytes))
}

func (m *Metrics) ObserveAsyncEvent(topic, event string) {
	m.asyncEvents.WithLabelValues(topic, event).Inc()
}

func mustRegisterCounterVec(registerer prometheus.Registerer, collector *prometheus.CounterVec) *prometheus.CounterVec {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}
		panic(err)
	}

	return collector
}

func mustRegisterHistogramVec(registerer prometheus.Registerer, collector *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing
			}
		}
		panic(err)
	}

	return collector
}
