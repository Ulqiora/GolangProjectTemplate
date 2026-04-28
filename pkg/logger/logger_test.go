package logger

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
)

func TestDefaultLogger(t *testing.T) {
	t.Parallel()

	log, err := NewLogger(EnvStage, otel.Tracer("test"))
	if err != nil {
		t.Fatalf("NewLogger returned error: %v", err)
	}

	SetDefaultLogger(log)
	if DefaultLogger() != log {
		t.Fatal("expected default logger to be set")
	}
}

func TestNewLoggerRejectsInvalidEnv(t *testing.T) {
	t.Parallel()

	if _, err := NewLogger(Env("bad"), nil); err == nil {
		t.Fatal("expected invalid env error")
	}
}

func TestLoggerWithSReturnsContext(t *testing.T) {
	t.Parallel()

	log, err := NewLogger(EnvStage, otel.Tracer("test"))
	if err != nil {
		t.Fatalf("NewLogger returned error: %v", err)
	}

	child, ctx := log.WithS(context.Background(), "child")
	if child == nil || ctx == nil {
		t.Fatal("expected child logger and context")
	}
	child.Stop()
}
