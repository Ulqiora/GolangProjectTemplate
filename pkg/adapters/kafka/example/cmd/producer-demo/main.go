package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/adapters/kafka/producer"
	transactional_producer "GolangTemplateProject/pkg/adapters/kafka/producer/transactional-producer"
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

func main() {
	rand.Seed(time.Now().UnixNano())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log, err := logger.NewLogger(logger.EnvStage, noop.NewTracerProvider().Tracer("kafka-example"))
	if err != nil {
		panic(err)
	}
	logger.SetDefaultLogger(log)

	cfg := producer.Config{
		Topic:           getEnv("KAFKA_TOPIC", "demo.events"),
		Brokers:         []string{getEnv("KAFKA_BROKER", "kafka:29092")},
		CompressionType: 1,
		ProduceSettings: producer.ProduceSettings{
			RequiredAcks:    -1,
			Idempotency:     true,
			TransactionalID: getEnv("KAFKA_TRANSACTIONAL_ID", "demo-producer-tx"),
			SaveReturningStatus: producer.SaveReturningStatus{
				Errors:    true,
				Succeeded: true,
			},
		},
	}

	txProducer, err := transactional_producer.NewTopicProducer[DemoEvent](cfg, log, nil)
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := txProducer.Close(); closeErr != nil {
			log.Error("Failed to close transactional producer", attribute.String("error", closeErr.Error()))
		}
	}()

	go runMetricsServer(log)
	go runProducerLoop(ctx, txProducer, log)

	<-ctx.Done()
	log.Info("Kafka producer demo stopped")
}

func runProducerLoop(ctx context.Context, producer kafka.TransactionalProducer[DemoEvent], log logger.Logger) {
	batchSize := getEnvInt("PRODUCER_BATCH_SIZE", 1000)
	sendInterval := getEnvDuration("PRODUCER_SEND_INTERVAL", 10*time.Millisecond)
	payloadMin := getEnvInt("PRODUCER_PAYLOAD_MIN_BYTES", 2048)
	payloadMax := getEnvInt("PRODUCER_PAYLOAD_MAX_BYTES", 8192)
	logEveryNBatches := getEnvInt("PRODUCER_LOG_EVERY_N_BATCHES", 20)

	if batchSize < 1 {
		batchSize = 1
	}
	if sendInterval <= 0 {
		sendInterval = 100 * time.Millisecond
	}
	if payloadMin < 1 {
		payloadMin = 1
	}
	if payloadMax < payloadMin {
		payloadMax = payloadMin
	}
	if logEveryNBatches < 1 {
		logEveryNBatches = 1
	}

	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	batchID := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			batchID++
			messages := make([]kafka.TypedMessage[DemoEvent], 0, batchSize)
			for i := 0; i < batchSize; i++ {
				payloadSize := payloadMin
				if payloadMax > payloadMin {
					payloadSize = payloadMin + rand.Intn(payloadMax-payloadMin+1)
				}

				event := DemoEvent{
					ID:        fmt.Sprintf("evt-%d-%d", batchID, i),
					Type:      "payment.authorized",
					UserID:    fmt.Sprintf("user-%d", rand.Intn(1000)+1),
					Amount:    float64(rand.Intn(50000)) / 100,
					Currency:  "RUB",
					CreatedAt: time.Now().UTC(),
					Payload:   randomString(payloadSize),
				}

				messages = append(messages, kafka.TypedMessage[DemoEvent]{
					Key:   event.UserID,
					Value: event,
					Headers: map[string]string{
						"source":       "kafka-example",
						"message_type": event.Type,
					},
				})
			}

			if err := producer.SendTypedMessagesTx(messages...); err != nil {
				log.Error(
					"Producer batch failed",
					attribute.String("error", err.Error()),
					attribute.Int("batch_id", batchID),
				)
				continue
			}

			if batchID%logEveryNBatches == 0 {
				log.Info(
					"Producer batch sent",
					attribute.Int("batch_id", batchID),
					attribute.Int("messages", len(messages)),
					attribute.String("send_interval", sendInterval.String()),
				)
			}
		}
	}
}

func runMetricsServer(log logger.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":2112"
	log.Info("Starting metrics server", attribute.String("addr", addr))
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("Metrics server stopped", attribute.String("error", err.Error()))
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

func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
