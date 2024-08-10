package open_telemetry

import (
	"context"
	"time"

	"GolangTemplateProject/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func newTraceProvider() (*trace.TracerProvider, error) {
	//res, err := resource.New(context.Background(),
	//	resource.WithAttributes(
	//		semconv.ServiceName("Route256"),
	//	),
	//)
	//traceExporter, err := otlptracehttp.New(context.Background(),
	//	otlptracehttp.WithInsecure(),
	//	otlptracehttp.WithEndpoint(config.Get().Trace.Jaeger.Connection.String()),
	//)
	////traceExporter, err := jaeger.New(
	////	jaeger.WithCollectorEndpoint(
	////		jaeger.WithEndpoint(config.Get().Trace.Jaeger.Connection.String()),
	////	),
	////)
	//if err != nil {
	//	return nil, err
	//}
	//
	//traceProvider := sdktrace.NewTracerProvider(
	//	sdktrace.WithSampler(sdktrace.AlwaysSample()),
	//	sdktrace.WithBatcher(traceExporter),
	//	sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(traceExporter)),
	//	sdktrace.WithResource(res),
	//)
	//traceExporter, err := otlptracehttp.New(context.Background())
	headers := map[string]string{
		"content-type": "application/json",
	}

	traceExporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(config.Get().Trace.Jaeger.Connection.String()),
			otlptracehttp.WithHeaders(headers),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(
			traceExporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("product-app"),
			),
		),
	)
	return traceProvider, nil

}
