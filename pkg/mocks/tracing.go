package mocks

import (
	"context"

	tracingpkg "GolangTemplateProject/pkg/smart-span/tracing"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	StartFunc func(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, tracingpkg.SmartSpan)
	Provider  trace.TracerProvider

	StartedNames []string
}

func (m *Tracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, tracingpkg.SmartSpan) {
	m.StartedNames = append(m.StartedNames, name)
	if m.StartFunc != nil {
		return m.StartFunc(ctx, name, opts...)
	}
	return ctx, trace.SpanFromContext(ctx)
}

func (m *Tracer) GetBaseTracer() trace.TracerProvider {
	return m.Provider
}

var _ tracingpkg.Tracer = (*Tracer)(nil)
