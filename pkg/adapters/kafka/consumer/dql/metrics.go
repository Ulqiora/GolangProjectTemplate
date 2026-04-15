package dql

import (
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const defaultMetricsNamespace = "dlq_consumer"

type consumerMetrics struct {
	activeSessions     *prometheus.GaugeVec
	sessionStarts      *prometheus.CounterVec
	sessionStops       *prometheus.CounterVec
	consumeRestarts    *prometheus.CounterVec
	batchesProcessed   *prometheus.CounterVec
	batchSize          *prometheus.HistogramVec
	batchDuration      *prometheus.HistogramVec
	messagesReceived   *prometheus.CounterVec
	messagesProcessed  *prometheus.CounterVec
	dlqWrites          *prometheus.CounterVec
	processingDuration *prometheus.HistogramVec
}

type consumerOptions struct {
	registerer prometheus.Registerer
	metrics    *consumerMetrics
	saveBatch  SaveBatchFunction
}

type ConsumerOption func(*consumerOptions)

func WithPrometheusRegisterer(registerer prometheus.Registerer) ConsumerOption {
	return func(options *consumerOptions) {
		options.registerer = registerer
	}
}

func WithConsumerMetrics(metrics *consumerMetrics) ConsumerOption {
	return func(options *consumerOptions) {
		options.metrics = metrics
	}
}

func WithDLQBatchSaver(saveBatch SaveBatchFunction) ConsumerOption {
	return func(options *consumerOptions) {
		options.saveBatch = saveBatch
	}
}

var (
	defaultMetricsOnce sync.Once
	defaultMetrics     *consumerMetrics
)

func resolveConsumerMetrics(options consumerOptions) *consumerMetrics {
	if options.metrics != nil {
		return options.metrics
	}
	if options.registerer != nil {
		return newConsumerMetrics(options.registerer)
	}

	defaultMetricsOnce.Do(func() {
		defaultMetrics = newConsumerMetrics(prometheus.DefaultRegisterer)
	})

	return defaultMetrics
}

func newConsumerMetrics(registerer prometheus.Registerer) *consumerMetrics {
	return &consumerMetrics{
		activeSessions: mustRegisterGaugeVec(registerer, prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "active_sessions",
				Help:      "Number of active Kafka consumer sessions.",
			},
			[]string{"group_id", "topic"},
		)),
		sessionStarts: mustRegisterCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "session_starts_total",
				Help:      "Total number of Kafka consumer group sessions started.",
			},
			[]string{"group_id", "topic"},
		)),
		sessionStops: mustRegisterCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "session_stops_total",
				Help:      "Total number of Kafka consumer group sessions stopped.",
			},
			[]string{"group_id", "topic", "status"},
		)),
		consumeRestarts: mustRegisterCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "restarts_total",
				Help:      "Total number of consumer loop restarts.",
			},
			[]string{"group_id", "topic", "reason"},
		)),
		batchesProcessed: mustRegisterCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "batches_processed_total",
				Help:      "Total number of processed batches.",
			},
			[]string{"group_id", "topic", "reason"},
		)),
		batchSize: mustRegisterHistogramVec(registerer, prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "batch_size",
				Help:      "Histogram of processed batch sizes.",
				Buckets:   []float64{1, 5, 10, 25, 50, 100, 200, 500, 1000},
			},
			[]string{"group_id", "topic"},
		)),
		batchDuration: mustRegisterHistogramVec(registerer, prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "batch_duration_seconds",
				Help:      "Batch processing duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"group_id", "topic", "reason"},
		)),
		messagesReceived: mustRegisterCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "messages_received_total",
				Help:      "Total number of Kafka messages received by the consumer.",
			},
			[]string{"group_id", "topic"},
		)),
		messagesProcessed: mustRegisterCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "messages_processed_total",
				Help:      "Total number of Kafka messages processed by outcome.",
			},
			[]string{"group_id", "topic", "status"},
		)),
		dlqWrites: mustRegisterCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "dlq_writes_total",
				Help:      "Total number of attempts to write a message to DLQ.",
			},
			[]string{"group_id", "topic", "status", "reason"},
		)),
		processingDuration: mustRegisterHistogramVec(registerer, prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: defaultMetricsNamespace,
				Name:      "processing_duration_seconds",
				Help:      "Kafka consumer message processing duration by outcome.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"group_id", "topic", "status"},
		)),
	}
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

func mustRegisterGaugeVec(registerer prometheus.Registerer, collector *prometheus.GaugeVec) *prometheus.GaugeVec {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(*prometheus.GaugeVec); ok {
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
