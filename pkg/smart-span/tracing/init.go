package tracing

import (
	"context"
	"crypto/tls"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

func InitTracing(ctx context.Context, cfg *TracerConfig) (*trace.TracerProvider, func(context.Context) error, error) {
	if cfg == nil {
		cfg = &TracerConfig{}
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithHeaders(cfg.Headers),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		otlptracehttp.WithTimeout(timeout),
	}

	if cfg.TLS.Enable {
		keyPath := cfg.TLS.KeyPath
		if keyPath == "" {
			keyPath = cfg.TLS.KayPath
		}

		certificate, err := tls.LoadX509KeyPair(cfg.TLS.CertificatePath, keyPath)
		if err != nil {
			return nil, nil, err
		}
		options = append(options,
			otlptracehttp.WithTLSClientConfig(&tls.Config{Certificates: []tls.Certificate{certificate}}),
		)
	} else {
		options = append(options,
			otlptracehttp.WithInsecure(),
		)
	}

	traceExporter, err := otlptrace.New(ctx,
		otlptracehttp.NewClient(options...),
	)
	if err != nil {
		return nil, nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(
			traceExporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(cfg.ServiceName),
			),
		),
	)
	otel.SetTracerProvider(tracerProvider)
	SetDefaultTracer(tracerProvider)
	if cfg.ServiceName != "" {
		SetDefaultServiceName(cfg.ServiceName)
	}
	return tracerProvider, tracerProvider.Shutdown, nil
}
