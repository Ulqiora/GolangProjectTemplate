package producer

import (
	"GolangTemplateProject/pkg/adapters/kafka"
	"github.com/prometheus/client_golang/prometheus"
)

type ErrorContext[T any] struct {
	Operation string
	Topic     string
	Message   *kafka.TypedMessage[T]
	Err       error
}

type ErrorHandler[T any] func(ErrorContext[T])

type RuntimeOptions[T any] struct {
	registerer     prometheus.Registerer
	metrics        *Metrics
	metricsEnabled bool
	errorHandler   ErrorHandler[T]
}

type ProducerOption[T any] func(*RuntimeOptions[T])

func WithPrometheusRegisterer[T any](registerer prometheus.Registerer) ProducerOption[T] {
	return func(options *RuntimeOptions[T]) {
		options.registerer = registerer
	}
}

func WithProducerMetrics[T any](metrics *Metrics) ProducerOption[T] {
	return func(options *RuntimeOptions[T]) {
		options.metrics = metrics
	}
}

func WithoutProducerMetrics[T any]() ProducerOption[T] {
	return func(options *RuntimeOptions[T]) {
		options.metricsEnabled = false
	}
}

func WithErrorHandler[T any](handler ErrorHandler[T]) ProducerOption[T] {
	return func(options *RuntimeOptions[T]) {
		options.errorHandler = handler
	}
}

func NewRuntimeOptions[T any](opts ...ProducerOption[T]) RuntimeOptions[T] {
	options := RuntimeOptions[T]{
		metricsEnabled: true,
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

func (o RuntimeOptions[T]) ResolveMetrics() *Metrics {
	if !o.metricsEnabled {
		return NoopProducerMetrics()
	}
	if o.metrics != nil {
		return o.metrics
	}
	if o.registerer != nil {
		return NewProducerMetrics(o.registerer)
	}
	return ResolveProducerMetrics()
}

func (o RuntimeOptions[T]) HandleError(operation, topic string, message *kafka.TypedMessage[T], err error) {
	if o.errorHandler == nil || err == nil {
		return
	}

	o.errorHandler(ErrorContext[T]{
		Operation: operation,
		Topic:     topic,
		Message:   message,
		Err:       err,
	})
}
