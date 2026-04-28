package transaction_manager

import (
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

const metricsNamespace = "transaction_manager"

type metrics struct {
	activeTransactions *prometheus.GaugeVec
	transactionsTotal  *prometheus.CounterVec
	transactionLatency *prometheus.HistogramVec
}

var (
	defaultMetricsOnce sync.Once
	defaultMetrics     *metrics
)

func resolveMetrics(registerer prometheus.Registerer) *metrics {
	if registerer == nil {
		defaultMetricsOnce.Do(func() {
			defaultMetrics = newMetrics(prometheus.DefaultRegisterer)
		})
		return defaultMetrics
	}

	return newMetrics(registerer)
}

func newMetrics(registerer prometheus.Registerer) *metrics {
	return &metrics{
		activeTransactions: mustRegisterGaugeVec(registerer, prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: metricsNamespace,
				Name:      "active_transactions",
				Help:      "Current number of active transactions handled by the transaction manager.",
			},
			[]string{"method", "access_mode", "isolation"},
		)),
		transactionsTotal: mustRegisterCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Name:      "transactions_total",
				Help:      "Total number of transactions handled by the transaction manager grouped by outcome.",
			},
			[]string{"method", "outcome", "access_mode", "isolation"},
		)),
		transactionLatency: mustRegisterHistogramVec(registerer, prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: metricsNamespace,
				Name:      "transaction_duration_seconds",
				Help:      "Duration of transaction manager operations grouped by outcome.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "outcome", "access_mode", "isolation"},
		)),
	}
}

func (m *metrics) IncActive(meta txMeta) {
	m.activeTransactions.WithLabelValues(meta.method, meta.accessMode, meta.isolation).Inc()
}

func (m *metrics) DecActive(meta txMeta) {
	m.activeTransactions.WithLabelValues(meta.method, meta.accessMode, meta.isolation).Dec()
}

func (m *metrics) Observe(meta txMeta, outcome string, startedAt time.Time) {
	m.transactionsTotal.WithLabelValues(meta.method, outcome, meta.accessMode, meta.isolation).Inc()
	m.transactionLatency.WithLabelValues(meta.method, outcome, meta.accessMode, meta.isolation).Observe(time.Since(startedAt).Seconds())
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

type txMeta struct {
	method     string
	accessMode string
	isolation  string
}

func newDoMeta() txMeta {
	return txMeta{
		method:     "do",
		accessMode: "read_write",
		isolation:  "default",
	}
}

func newDoxMeta(opts pgx.TxOptions) txMeta {
	return txMeta{
		method:     "dox",
		accessMode: accessModeString(opts.AccessMode),
		isolation:  isolationLevelString(opts.IsoLevel),
	}
}

func accessModeString(mode pgx.TxAccessMode) string {
	switch mode {
	case pgx.ReadOnly:
		return "read_only"
	case pgx.ReadWrite:
		return "read_write"
	default:
		return "default"
	}
}

func isolationLevelString(level pgx.TxIsoLevel) string {
	switch level {
	case pgx.Serializable:
		return "serializable"
	case pgx.RepeatableRead:
		return "repeatable_read"
	case pgx.ReadCommitted:
		return "read_committed"
	case pgx.ReadUncommitted:
		return "read_uncommitted"
	default:
		return "default"
	}
}
