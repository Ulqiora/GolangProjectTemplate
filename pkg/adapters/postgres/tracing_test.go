package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestPGXTracerCreatesChildQuerySpan(t *testing.T) {
	exporter := &postgresSpanExporter{}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)))
	defer provider.Shutdown(context.Background())

	tracer := newPGXTracer(provider, "master", EndpointConfig{
		Host:     "postgres",
		Port:     "5432",
		Database: "postgres",
		User:     "postgres",
	})

	ctx, root := provider.Tracer("postgres-test").Start(context.Background(), "request")
	ctx = tracer.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "select 1"})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
	root.End()

	if err := provider.ForceFlush(context.Background()); err != nil {
		t.Fatalf("force flush: %v", err)
	}

	querySpan := findPostgresSpan(exporter.Spans(), "postgres.query")
	if querySpan == nil {
		t.Fatalf("postgres.query span not found; got %v", postgresSpanNames(exporter.Spans()))
	}
	if querySpan.SpanContext().TraceID() != root.SpanContext().TraceID() {
		t.Fatalf("query trace id = %s, want %s", querySpan.SpanContext().TraceID(), root.SpanContext().TraceID())
	}
	if querySpan.Parent().SpanID() != root.SpanContext().SpanID() {
		t.Fatalf("query parent = %s, want %s", querySpan.Parent().SpanID(), root.SpanContext().SpanID())
	}
}

func TestPGXTracerMarksQueryErrors(t *testing.T) {
	exporter := &postgresSpanExporter{}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)))
	defer provider.Shutdown(context.Background())

	tracer := newPGXTracer(provider, "master", EndpointConfig{})
	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: "select 1"})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: errors.New("boom")})

	if err := provider.ForceFlush(context.Background()); err != nil {
		t.Fatalf("force flush: %v", err)
	}

	querySpan := findPostgresSpan(exporter.Spans(), "postgres.query")
	if querySpan == nil {
		t.Fatal("postgres.query span not found")
	}
	if querySpan.Status().Code.String() != "Error" {
		t.Fatalf("query status = %s, want Error", querySpan.Status().Code.String())
	}
}

func TestWithTracingOptionUsesProvidedTracerProvider(t *testing.T) {
	provider := sdktrace.NewTracerProvider()
	defer provider.Shutdown(context.Background())

	opts := buildOptions(WithTracing(WithTracerProvider(provider)))
	if !opts.tracingEnabled {
		t.Fatal("expected tracing to be enabled")
	}
	if opts.tracerProvider != provider {
		t.Fatal("expected configured tracer provider")
	}

	defaultOpts := buildOptions()
	if defaultOpts.tracingEnabled {
		t.Fatal("expected tracing to be disabled by default")
	}
	if defaultOpts.tracerProvider != otel.GetTracerProvider() {
		t.Fatal("expected default tracer provider to be resolved")
	}
}

type postgresSpanExporter struct {
	spans []sdktrace.ReadOnlySpan
}

func (e *postgresSpanExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *postgresSpanExporter) Shutdown(context.Context) error { return nil }

func (e *postgresSpanExporter) Spans() []sdktrace.ReadOnlySpan {
	spans := make([]sdktrace.ReadOnlySpan, len(e.spans))
	copy(spans, e.spans)
	return spans
}

func findPostgresSpan(spans []sdktrace.ReadOnlySpan, name string) sdktrace.ReadOnlySpan {
	for _, span := range spans {
		if span.Name() == name {
			return span
		}
	}
	return nil
}

func postgresSpanNames(spans []sdktrace.ReadOnlySpan) []string {
	names := make([]string, 0, len(spans))
	for _, span := range spans {
		names = append(names, span.Name())
	}
	return names
}

var _ trace.TracerProvider = (*sdktrace.TracerProvider)(nil)
