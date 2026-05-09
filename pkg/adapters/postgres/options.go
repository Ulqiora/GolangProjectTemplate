package postgres

import (
	"errors"

	"GolangTemplateProject/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var ErrNilConfig = errors.New("postgres config is nil")

type Option func(*options)

type options struct {
	log            logger.Logger
	registerer     prometheus.Registerer
	tracingEnabled bool
	tracerProvider trace.TracerProvider
}

type TracingOption func(*options)

func WithLogger(log logger.Logger) Option {
	return func(opts *options) {
		opts.log = log
	}
}

func WithRegisterer(registerer prometheus.Registerer) Option {
	return func(opts *options) {
		opts.registerer = registerer
	}
}

func WithTracing(traceOpts ...TracingOption) Option {
	return func(opts *options) {
		opts.tracingEnabled = true
		opts.tracerProvider = otel.GetTracerProvider()
		for _, traceOpt := range traceOpts {
			if traceOpt != nil {
				traceOpt(opts)
			}
		}
	}
}

func WithTracerProvider(provider trace.TracerProvider) TracingOption {
	return func(opts *options) {
		if provider != nil {
			opts.tracerProvider = provider
		}
	}
}

func buildOptions(opts ...Option) options {
	resolved := options{
		log:            resolveLogger(nil),
		registerer:     prometheus.DefaultRegisterer,
		tracerProvider: otel.GetTracerProvider(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&resolved)
		}
	}

	resolved.log = resolveLogger(resolved.log)

	return resolved
}

func resolveLogger(log logger.Logger) logger.Logger {
	if log != nil {
		return log
	}
	if defaultLogger := logger.DefaultLogger(); defaultLogger != nil {
		return defaultLogger
	}

	fallback, err := logger.NewLogger(logger.EnvStage, nil)
	if err != nil {
		panic(err)
	}

	return fallback
}
