package tracing

import (
	"context"

	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/smart-span/stacktrace"
	"go.opentelemetry.io/otel/trace"
)

type Tracer interface {
	Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, SmartSpan)
	GetBaseTracer() trace.TracerProvider
}

type TracerBase struct {
	tracer trace.TracerProvider
	logger logger.Logger
}

func NewTracer(tracer trace.TracerProvider, logger logger.Logger) Tracer {
	return &TracerBase{
		tracer: tracer,
		logger: logger,
	}
}

func (t *TracerBase) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, SmartSpan) {
	if name == "" {
		name = stacktrace.TakeOnceCalledFunction()
	}
	ctx, span := t.tracer.Tracer(defaultServiceName).Start(ctx, name, opts...)
	return ctx, &SmartSpanBase{
		spanBase: span,
		logger:   t.logger,
	}
}

func (t *TracerBase) GetBaseTracer() trace.TracerProvider {
	return t.tracer
}
