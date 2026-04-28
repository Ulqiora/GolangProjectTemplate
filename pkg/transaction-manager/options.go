package transaction_manager

import (
	"GolangTemplateProject/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
)

type Option func(*options)

type options struct {
	log        logger.Logger
	registerer prometheus.Registerer
}

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

func buildOptions(opts ...Option) options {
	resolved := options{
		log:        resolveLogger(nil),
		registerer: prometheus.DefaultRegisterer,
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
