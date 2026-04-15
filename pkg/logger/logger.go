package logger

import (
	"context"
	"errors"

	"GolangTemplateProject/pkg/logger/attribute"
	"GolangTemplateProject/pkg/smart-span/stacktrace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Env string

const (
	EnvDev   Env = "Dev"
	EnvStage Env = "Stage"
	EnvProd  Env = "Prod"
)

const (
	ProjectLoggerCtxKey = "project_logger"
)

var (
	defaultLogger Logger
)

func SetDefaultLogger(logger Logger) {
	defaultLogger = logger
}

func DefaultLogger() Logger {
	return defaultLogger
}

type Logger interface {
	Debug(msg string, fields ...attribute.Field)
	Info(msg string, fields ...attribute.Field)
	Warn(msg string, fields ...attribute.Field)
	Error(msg string, fields ...attribute.Field)
	DPanic(msg string, fields ...attribute.Field)
	Panic(msg string, fields ...attribute.Field)
	Fatal(msg string, fields ...attribute.Field)
	Sync() error
	Stop()
	With(fields ...attribute.Field) Logger
	WithN(name string, fields ...attribute.Field) Logger
	WithS(ctx context.Context, name string, fields ...attribute.Field) (Logger, context.Context)
}

type loggerImpl struct {
	name       string
	loggerBase *zap.Logger
	tracer     trace.Tracer
	span       trace.Span
}

func (t loggerImpl) Debug(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, attribute.ToZapField(f))
	}
	t.loggerBase.Debug(msg, zapFields...)
	if t.span != nil {
		t.span.AddEvent(msg, trace.WithAttributes(
			attribute.ToOtelAttributes(fields)...,
		))
	}
}

func (t loggerImpl) Info(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, attribute.ToZapField(f))
	}
	t.loggerBase.Info(msg, zapFields...)
	if t.span != nil {
		t.span.AddEvent(msg, trace.WithAttributes(
			attribute.ToOtelAttributes(fields)...,
		))
	}
}

func (t loggerImpl) Warn(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, attribute.ToZapField(f))
	}
	t.loggerBase.Warn(msg, zapFields...)
	if t.span != nil {
		t.span.AddEvent(msg, trace.WithAttributes(
			attribute.ToOtelAttributes(fields)...,
		))
	}
}

func (t loggerImpl) Error(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, attribute.ToZapField(f))
	}
	t.loggerBase.Error(msg, zapFields...)
	if t.span != nil {
		t.span.AddEvent(msg, trace.WithAttributes(
			attribute.ToOtelAttributes(fields)...,
		))
	}
}

func (t loggerImpl) DPanic(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, attribute.ToZapField(f))
	}
	t.loggerBase.DPanic(msg, zapFields...)
	if t.span != nil {
		t.span.AddEvent(msg, trace.WithAttributes(
			attribute.ToOtelAttributes(fields)...,
		))
	}
}

func (t loggerImpl) Panic(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, attribute.ToZapField(f))
	}
	t.loggerBase.Panic(msg, zapFields...)
	if t.span != nil {
		t.span.AddEvent(msg, trace.WithAttributes(
			attribute.ToOtelAttributes(fields)...,
		))
	}
}

func (t loggerImpl) Fatal(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, attribute.ToZapField(f))
	}
	t.loggerBase.Fatal(msg, zapFields...)
	if t.span != nil {
		t.span.AddEvent(msg, trace.WithAttributes(
			attribute.ToOtelAttributes(fields)...,
		))
	}
}

func (t loggerImpl) Sync() error {
	return t.loggerBase.Sync()
}

func (t loggerImpl) Stop() {
	t.span.End()
}

func (t loggerImpl) With(fields ...attribute.Field) Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, attribute.ToZapField(f))
	}
	logger := t.loggerBase.With(zapFields...)
	return loggerImpl{
		name:       t.name,
		tracer:     t.tracer,
		loggerBase: logger,
	}
}

func (t loggerImpl) WithN(name string, fields ...attribute.Field) Logger {
	name = t.name + ":" + name
	fields = append(fields, attribute.String("name", name))
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, attribute.ToZapField(f))
	}
	logger := t.loggerBase.With(zapFields...)
	return loggerImpl{
		name:       t.name + ":" + name,
		tracer:     t.tracer,
		loggerBase: logger,
	}
}

func (t loggerImpl) WithS(ctx context.Context, name string, fields ...attribute.Field) (Logger, context.Context) {
	name = t.name + ":" + name
	fields = append(fields, attribute.String("name", name))
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, attribute.ToZapField(f))
	}
	logger := t.loggerBase.With(zapFields...)
	ctx, span := t.tracer.Start(
		ctx,
		stacktrace.TakeOnceCalledFunction(),
		trace.WithAttributes(
			attribute.ToOtelAttributes(fields)...,
		),
	)
	return loggerImpl{
		name:       name,
		tracer:     t.tracer,
		loggerBase: logger,
		span:       span,
	}, ctx
}

func NewLogger(env Env, tracer trace.Tracer) (Logger, error) {
	logger, err := envTologger(env)
	return loggerImpl{
		tracer:     tracer,
		loggerBase: logger,
	}, err
}

func envTologger(env Env) (*zap.Logger, error) {
	switch env {
	case EnvDev:
	case EnvStage:
		return zap.NewDevelopment()
	case EnvProd:
		return zap.NewProduction()
	}
	return nil, errors.New("invalid environment")
}
