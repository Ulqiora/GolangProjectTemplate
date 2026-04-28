package tracing

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestRecordErrorNilDoesNotPanic(t *testing.T) {
	t.Parallel()

	tracerProvider := sdktrace.NewTracerProvider()
	tracer := NewTracer(tracerProvider, nil)
	_, span := tracer.Start(context.Background(), "record-error-nil")

	span.RecordError(nil)
	span.End()
}

func TestTracerProviderReturnsSpanProvider(t *testing.T) {
	t.Parallel()

	tracerProvider := sdktrace.NewTracerProvider()
	tracer := NewTracer(tracerProvider, nil)
	_, span := tracer.Start(context.Background(), "provider")
	defer span.End()

	if got := span.TracerProvider(); got != tracerProvider {
		t.Fatalf("expected span tracer provider %p, got %p", tracerProvider, got)
	}
}

func TestNewTracerFallsBackToGlobalProvider(t *testing.T) {
	t.Parallel()

	tracer := NewTracer(nil, nil)
	_, span := tracer.Start(context.Background(), "fallback-provider")
	span.RecordError(errors.New("boom"))
	span.End()

	if tracer.GetBaseTracer() == nil {
		t.Fatal("expected fallback tracer provider to be configured")
	}
}

func TestTracerStart_EmptyName_UsesCallerFunction(t *testing.T) {
	t.Parallel()

	exp := &inMemoryExporter{}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
	)
	tracer := NewTracer(tracerProvider, nil)

	_, span := helperStartSpanWithEmptyName(t, tracer)
	span.End()

	name := exp.lastSpanName()
	if name == "" {
		t.Fatalf("expected auto-generated span name, got empty")
	}
	if strings.Contains(name, "\n") {
		t.Fatalf("expected single-line span name, got %q", name)
	}
	if !strings.Contains(name, "helperStartSpanWithEmptyName") {
		t.Fatalf("expected span name to include caller function, got %q", name)
	}
}

func helperStartSpanWithEmptyName(t *testing.T, tracer Tracer) (context.Context, SmartSpan) {
	t.Helper()
	return tracer.Start(context.Background(), "")
}

type inMemoryExporter struct {
	mu    sync.Mutex
	names []string
}

func (e *inMemoryExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, s := range spans {
		e.names = append(e.names, s.Name())
	}
	return nil
}

func (e *inMemoryExporter) Shutdown(context.Context) error { return nil }

func (e *inMemoryExporter) lastSpanName() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.names) == 0 {
		return ""
	}
	return e.names[len(e.names)-1]
}
