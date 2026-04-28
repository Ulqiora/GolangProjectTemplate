package mocks

import (
	"context"

	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
)

type Logger struct {
	Messages []string
	Fields   [][]attribute.Field

	SyncErr error

	WithCalls  int
	WithNCalls int
	WithSCalls int
	StopCalls  int
}

func (m *Logger) Debug(msg string, fields ...attribute.Field)  { m.record(msg, fields...) }
func (m *Logger) Info(msg string, fields ...attribute.Field)   { m.record(msg, fields...) }
func (m *Logger) Warn(msg string, fields ...attribute.Field)   { m.record(msg, fields...) }
func (m *Logger) Error(msg string, fields ...attribute.Field)  { m.record(msg, fields...) }
func (m *Logger) DPanic(msg string, fields ...attribute.Field) { m.record(msg, fields...) }
func (m *Logger) Panic(msg string, fields ...attribute.Field)  { m.record(msg, fields...) }
func (m *Logger) Fatal(msg string, fields ...attribute.Field)  { m.record(msg, fields...) }

func (m *Logger) Sync() error { return m.SyncErr }

func (m *Logger) Stop() { m.StopCalls++ }

func (m *Logger) With(fields ...attribute.Field) logger.Logger {
	m.WithCalls++
	return m
}

func (m *Logger) WithN(name string, fields ...attribute.Field) logger.Logger {
	m.WithNCalls++
	return m
}

func (m *Logger) WithS(ctx context.Context, name string, fields ...attribute.Field) (logger.Logger, context.Context) {
	m.WithSCalls++
	return m, ctx
}

func (m *Logger) record(msg string, fields ...attribute.Field) {
	m.Messages = append(m.Messages, msg)
	m.Fields = append(m.Fields, fields)
}
