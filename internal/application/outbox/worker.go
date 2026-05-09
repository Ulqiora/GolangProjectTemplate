package outbox

import (
	"context"
	"fmt"
	"time"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	smarttracing "GolangTemplateProject/pkg/smart-span/tracing"
	transactionmanager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/prometheus/client_golang/prometheus"
	otelattribute "go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Worker struct {
	txManager transactionmanager.TransactionManager
	repo      ports.OutboxRepository
	publisher ports.OutboxEventPublisher
	log       logger.Logger
	batchSize uint
	interval  time.Duration
	metrics   *metrics
}

func NewWorker(
	txManager transactionmanager.TransactionManager,
	repo ports.OutboxRepository,
	publisher ports.OutboxEventPublisher,
	log logger.Logger,
	registerer prometheus.Registerer,
) *Worker {
	baseLogger := log
	if baseLogger == nil {
		baseLogger = logger.DefaultLogger()
	}
	if baseLogger == nil {
		fallback, err := logger.NewLogger(logger.EnvStage, nil)
		if err != nil {
			panic(err)
		}
		baseLogger = fallback
	}

	return &Worker{
		txManager: txManager,
		repo:      repo,
		publisher: publisher,
		log:       baseLogger.WithN("outbox_worker"),
		batchSize: 50,
		interval:  2 * time.Second,
		metrics:   newMetrics(registerer),
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil {
				w.log.Error("Outbox worker iteration failed", attribute.String("error", err.Error()))
			}
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	ctx, span := smarttracing.GetDefaultTracer().Start(ctx, "auth.outbox.run_once", trace.WithAttributes(
		otelattribute.Int("auth.outbox.batch_size", int(w.batchSize)),
	))
	defer span.End()

	startedAt := time.Now()

	var messages []*domain.OutboxMessage
	err := w.txManager.Do(ctx, func(txCtx context.Context) error {
		picked, pickErr := w.repo.PickPending(txCtx, w.batchSize)
		if pickErr != nil {
			return pickErr
		}

		for _, message := range picked {
			if markErr := w.repo.MarkProcessing(txCtx, message.ID.String()); markErr != nil {
				return fmt.Errorf("mark outbox message as processing: %w", markErr)
			}
			message.Status = domain.OutboxStatusProcessing
			message.NextAttemptAt = nil
		}

		messages = picked
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		w.metrics.observe("pick_error", startedAt)
		return err
	}
	if len(messages) == 0 {
		span.SetStatus(otelcodes.Ok, "idle")
		w.metrics.observe("idle", startedAt)
		return nil
	}

	for _, message := range messages {
		err := w.publisher.Publish(ctx, message)
		if err != nil {
			span.RecordError(err, trace.WithAttributes(
				otelattribute.String("auth.outbox.message_id", message.ID.String()),
				otelattribute.String("auth.outbox.topic", message.Topic),
			))
			nextAttemptAt := time.Now().UTC().Add(retryBackoff(message.AttemptCount + 1))
			_ = w.txManager.Do(ctx, func(txCtx context.Context) error {
				return w.repo.MarkFailed(txCtx, message.ID.String(), err.Error(), nextAttemptAt)
			})
			w.metrics.observe("publish_error", startedAt)
			continue
		}

		if err := w.txManager.Do(ctx, func(txCtx context.Context) error {
			return w.repo.MarkSent(txCtx, message.ID.String())
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			w.metrics.observe("mark_sent_error", startedAt)
			return err
		}
	}

	span.SetAttributes(otelattribute.Int("auth.outbox.processed_messages", len(messages)))
	span.SetStatus(otelcodes.Ok, "processed")
	w.metrics.observe("success", startedAt)
	return nil
}

func retryBackoff(attempt int32) time.Duration {
	switch {
	case attempt <= 1:
		return 5 * time.Second
	case attempt == 2:
		return 15 * time.Second
	case attempt == 3:
		return 30 * time.Second
	case attempt <= 5:
		return time.Minute
	default:
		return 5 * time.Minute
	}
}

type metrics struct {
	iterations *prometheus.CounterVec
	duration   *prometheus.HistogramVec
}

func newMetrics(registerer prometheus.Registerer) *metrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	iterations := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "auth_outbox",
			Name:      "iterations_total",
			Help:      "Total number of outbox worker iterations by status.",
		},
		[]string{"status"},
	)
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "auth_outbox",
			Name:      "iteration_duration_seconds",
			Help:      "Outbox worker iteration duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"status"},
	)

	if err := registerer.Register(iterations); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.CounterVec); ok {
				iterations = existing
			}
		}
	}
	if err := registerer.Register(duration); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.HistogramVec); ok {
				duration = existing
			}
		}
	}

	return &metrics{iterations: iterations, duration: duration}
}

func (m *metrics) observe(status string, startedAt time.Time) {
	m.iterations.WithLabelValues(status).Inc()
	m.duration.WithLabelValues(status).Observe(time.Since(startedAt).Seconds())
}
