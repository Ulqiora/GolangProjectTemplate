package tracing

import (
	"context"

	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/smart-span/stacktrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	defaultServiceName = "default"
	defaultTracer      = NewTracer(otel.GetTracerProvider(), logger.DefaultLogger())
)

func SetDefaultServiceName(name string) {
	defaultServiceName = name
}
func SetDefaultTracer(tracer trace.TracerProvider) {
	defaultTracer = NewTracer(tracer, logger.DefaultLogger())
}

func GetDefaultTracer() Tracer {
	return defaultTracer
}

type SmartSpan interface {
	End(options ...trace.SpanEndOption)
	AddEvent(name string, options ...trace.EventOption)
	AddLink(link trace.Link)
	IsRecording() bool
	RecordError(err error, options ...trace.EventOption)
	SpanContext() trace.SpanContext
	SetStatus(code codes.Code, description string)
	SetName(name string)
	SetAttributes(kv ...attribute.KeyValue)
	TracerProvider() trace.TracerProvider
}

type SmartSpanBase struct {
	spanBase trace.Span
	logger   logger.Logger
}

func (s SmartSpanBase) End(options ...trace.SpanEndOption) {
	s.spanBase.End(options...)
}

func (s SmartSpanBase) AddEvent(name string, options ...trace.EventOption) {
	s.spanBase.AddEvent(name, options...)
	//trace.WithSchemaURL()
	if s.logger == nil {
		(s.logger).Info(name)
	}
}

func (s SmartSpanBase) AddLink(link trace.Link) {
	s.spanBase.AddLink(link)
}

func (s SmartSpanBase) IsRecording() bool {
	return s.spanBase.IsRecording()
}

func (s SmartSpanBase) RecordError(err error, options ...trace.EventOption) {
	s.spanBase.RecordError(err, options...)
	if s.logger != nil {
		(s.logger).Error(err.Error())
	}
}

func (s SmartSpanBase) SpanContext() trace.SpanContext {
	return s.spanBase.SpanContext()
}

func (s SmartSpanBase) SetStatus(code codes.Code, description string) {
	s.spanBase.SetStatus(code, description)
}

func (s SmartSpanBase) SetName(name string) {
	s.spanBase.SetName(name)
}

func (s SmartSpanBase) SetAttributes(kv ...attribute.KeyValue) {
	s.spanBase.SetAttributes(kv...)
}

func (s SmartSpanBase) TracerProvider() trace.TracerProvider {
	return defaultTracer.GetBaseTracer()
}

type SmartSpanBuilder struct {
	name   string
	logger logger.Logger
}

func SetName(name string) SmartSpanBuilder {
	var span SmartSpanBuilder
	if name == "" {
		span.name = stacktrace.TakeOnceCalledFunction()
	} else {
		span.name = name
	}
	return span
}

func (b *SmartSpanBuilder) Start(ctx context.Context) (context.Context, SmartSpanBase) {
	ctx, span := defaultTracer.GetBaseTracer().Tracer(defaultServiceName).Start(ctx, b.name)
	return ctx, SmartSpanBase{
		spanBase: span,
		logger:   b.logger,
	}
}
