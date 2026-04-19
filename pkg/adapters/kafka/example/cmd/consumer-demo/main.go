package main

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"GolangTemplateProject/pkg/adapters/kafka/consumer/dql"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace/noop"
)

type DemoEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	UserID    string    `json:"user_id"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	Payload   string    `json:"payload"`
}

func (e *DemoEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *DemoEvent) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, &e)
}

func (e *DemoEvent) Params() map[string]interface{} {
	return map[string]interface{}{
		"id":         e.ID,
		"type":       e.Type,
		"user_id":    e.UserID,
		"amount":     e.Amount,
		"currency":   e.Currency,
		"created_at": e.CreatedAt,
	}
}

func (e *DemoEvent) Fields() []string {
	return []string{"id", "type", "user_id", "amount", "currency", "created_at"}
}

func (e *DemoEvent) PrimaryKey() (string, any) {
	return "id", e.ID
}

func main() {
	rand.Seed(time.Now().UnixNano())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log, err := logger.NewLogger(logger.EnvStage, noop.NewTracerProvider().Tracer("kafka-consumer-example"))
	if err != nil {
		panic(err)
	}
	logger.SetDefaultLogger(log)

	batchSize := getEnvInt("CONSUMER_BATCH_SIZE", 1000)
	batchTimeout := getEnvDuration("CONSUMER_BATCH_TIMEOUT", 200*time.Millisecond)
	dlqSave := getEnvBool("CONSUMER_DLQ_SAVE", true)
	dlqRetries := getEnvInt("CONSUMER_DLQ_RETRIES", 3)
	dlqTimeout := getEnvDuration("CONSUMER_DLQ_TIMEOUT", 2*time.Second)
	logEveryNMessages := getEnvInt("CONSUMER_LOG_EVERY_N_MESSAGES", 5000)
	demoFailurePercent = getEnvInt("CONSUMER_FAILURE_PERCENT", 1)
	if batchSize < 1 {
		batchSize = 1
	}
	if batchTimeout <= 0 {
		batchTimeout = 200 * time.Millisecond
	}
	if logEveryNMessages < 1 {
		logEveryNMessages = 1
	}
	if dlqRetries < 1 {
		dlqRetries = 1
	}
	if dlqTimeout <= 0 {
		dlqTimeout = 2 * time.Second
	}
	if demoFailurePercent < 0 {
		demoFailurePercent = 0
	}
	if demoFailurePercent > 100 {
		demoFailurePercent = 100
	}

	cfg := &dql.Config{
		Topic:   getEnv("KAFKA_TOPIC", "demo.events"),
		Brokers: dql.Brokers{getEnv("KAFKA_BROKER", "kafka:29092")},
		GroupSettings: dql.GroupSettings{
			RebalancedGroupStrategy: "sticky",
			ReturnErrors:            true,
			OffsetInitial:           "new",
			IsolationLevel:          "committed",
			GroupID:                 getEnv("KAFKA_GROUP_ID", "demo-consumer-group"),
		},
		ConsumeSettings: dql.ConsumeProcessConfig{
			DlqSave:            dlqSave,
			DlqRetries:         int8(dlqRetries),
			DlqTimeout:         dlqTimeout,
			BatchEnabled:       true,
			BatchSize:          batchSize,
			BatchTimeout:       batchTimeout,
			PerMessageDebugLog: false,
			LogEveryNMessages:  logEveryNMessages,
		},
	}

	consumer, err := dql.NewTopicConsumerGroupDlq[*DemoEvent](
		cfg,
		log,
		func(_ context.Context, _ *dql.DLQMessage) error { return nil },
		execDemoEvent,
	)
	if err != nil {
		panic(err)
	}

	go runMetricsServer(log)
	consumer.Run(ctx)
	log.Info("Kafka consumer demo started", attribute.String("topic", cfg.Topic), attribute.String("group_id", cfg.GroupSettings.GroupID))

	<-ctx.Done()

	select {
	case <-consumer.WaitStoppedSession():
		log.Info("Kafka consumer demo stopped gracefully")
	case <-time.After(15 * time.Second):
		log.Warn("Kafka consumer demo stop timeout")
	}
}

var demoFailurePercent = 1

func execDemoEvent(_ context.Context, event *DemoEvent, _ *dql.MapValues) error {
	// Simulate occasional processing failure to demonstrate failed metrics and DLQ flow.
	if event == nil {
		return nil
	}

	if rand.Intn(100) < demoFailurePercent {
		return errProcessingFailed
	}

	return nil
}

var errProcessingFailed = &processingError{message: "demo processing failed"}

type processingError struct {
	message string
}

func (e *processingError) Error() string {
	return e.message
}

func runMetricsServer(log logger.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":2113"
	log.Info("Starting consumer metrics server", attribute.String("addr", addr))
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("Consumer metrics server stopped", attribute.String("error", err.Error()))
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}

	return parsed
}
