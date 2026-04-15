//go:build integration
// +build integration

package dql

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

type paymentMessage struct {
	PaymentID        string            `json:"payment_id"`
	ExternalID       string            `json:"external_id"`
	UserID           string            `json:"user_id"`
	MerchantID       string            `json:"merchant_id"`
	OrderID          string            `json:"order_id"`
	AmountMinor      int64             `json:"amount_minor"`
	FeeMinor         int64             `json:"fee_minor"`
	Currency         string            `json:"currency"`
	Method           string            `json:"method"`
	Status           string            `json:"status"`
	Country          string            `json:"country"`
	Provider         string            `json:"provider"`
	CreatedAt        time.Time         `json:"created_at"`
	AuthorizedAt     time.Time         `json:"authorized_at"`
	Description      string            `json:"description"`
	CustomerEmail    string            `json:"customer_email"`
	CustomerPhone    string            `json:"customer_phone"`
	IpAddress        string            `json:"ip_address"`
	IdempotencyKey   string            `json:"idempotency_key"`
	CardMask         string            `json:"card_mask"`
	DeviceID         string            `json:"device_id"`
	SessionID        string            `json:"session_id"`
	ReferenceCode    string            `json:"reference_code"`
	Metadata         map[string]string `json:"metadata"`
	LineItems        []paymentLineItem `json:"line_items"`
	BillingAddress   address           `json:"billing_address"`
	ShippingAddress  address           `json:"shipping_address"`
	RiskSignals      riskSignals       `json:"risk_signals"`
	AdditionalFields map[string]string `json:"additional_fields"`
}

type paymentLineItem struct {
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Quantity    int32  `json:"quantity"`
	AmountMinor int64  `json:"amount_minor"`
	Category    string `json:"category"`
}

type address struct {
	Country    string `json:"country"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Street     string `json:"street"`
	Building   string `json:"building"`
}

type riskSignals struct {
	IsVpn             bool    `json:"is_vpn"`
	IsProxy           bool    `json:"is_proxy"`
	ChargebackRate    float64 `json:"chargeback_rate"`
	VelocityPerHour   int32   `json:"velocity_per_hour"`
	VelocityPerDay    int32   `json:"velocity_per_day"`
	DeviceTrustScore  float64 `json:"device_trust_score"`
	EmailTrustScore   float64 `json:"email_trust_score"`
	PhoneTrustScore   float64 `json:"phone_trust_score"`
	MerchantRiskScore float64 `json:"merchant_risk_score"`
}

func (m *paymentMessage) Params() map[string]interface{} {
	return map[string]interface{}{
		"payment_id":   m.PaymentID,
		"external_id":  m.ExternalID,
		"user_id":      m.UserID,
		"merchant_id":  m.MerchantID,
		"order_id":     m.OrderID,
		"amount_minor": m.AmountMinor,
		"currency":     m.Currency,
		"status":       m.Status,
	}
}

func (m *paymentMessage) Fields() []string {
	return []string{"payment_id", "external_id", "user_id", "merchant_id", "order_id", "amount_minor", "currency", "status"}
}

func (m *paymentMessage) PrimaryKey() (string, any) {
	return "payment_id", m.PaymentID
}

func (m *paymentMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *paymentMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}

func TestConsumerPerformancePayments(t *testing.T) {
	if testing.Short() {
		t.Skip("integration performance test is skipped in short mode")
	}

	env := newKafkaIntegrationEnv(t, 5*time.Minute, "dql-payment-integration-test")
	topic, groupID := env.newTopicAndGroup("dlq-consumer-payments")
	env.createTopic(topic)

	registry := prometheus.NewRegistry()
	const messagesTotal int64 = 3000

	processed := atomic.Int64{}
	done := make(chan struct{})

	consumer, err := NewTopicConsumerGroupDlq[*paymentMessage](
		&Config{
			Topic:   topic,
			Brokers: env.brokers,
			GroupSettings: GroupSettings{
				GroupID:                 groupID,
				OffsetInitial:           "old",
				RebalancedGroupStrategy: "round-robin",
				IsolationLevel:          "commited",
				ReturnErrors:            true,
			},
			ConsumeSettings: ConsumeProcessConfig{
				PerMessageDebugLog: false,
				LogEveryNMessages:  1000,
				BatchEnabled:       true,
				BatchSize:          100,
				BatchTimeout:       200 * time.Millisecond,
			},
		},
		env.logger,
		func(ctx context.Context, object *DLQMessage) error {
			return fmt.Errorf("unexpected DLQ write for message %s", object.ID)
		},
		func(ctx context.Context, object *paymentMessage, values *MapValues) error {
			if err := validatePayment(object); err != nil {
				return err
			}

			normalizedEmail := strings.ToLower(strings.TrimSpace(object.CustomerEmail))
			normalizedDescription := strings.TrimSpace(strings.ToUpper(object.Description))
			feeRate := float64(object.FeeMinor) / float64(object.AmountMinor)
			avgLineItemAmount := averageLineItemAmount(object.LineItems)
			riskScore := computeRiskScore(object.RiskSignals, feeRate, avgLineItemAmount)
			checksum := buildPaymentChecksum(object, normalizedEmail, normalizedDescription, riskScore)
			if len(checksum) != 64 {
				return fmt.Errorf("invalid checksum length")
			}

			if processed.Add(1) == messagesTotal {
				close(done)
			}
			return nil
		},
		WithPrometheusRegisterer(registry),
	)
	require.NoError(t, err)

	consumerCancel := env.startConsumer(consumer)
	defer stopConsumerGracefully(t, consumer, consumerCancel)

	producer := env.newProducer()
	messages := buildPaymentMessages(t, topic, messagesTotal)

	startedAt := time.Now()
	sendMessages(t, producer, messages)
	waitUntilDone(
		t,
		done,
		90*time.Second,
		"timeout waiting for consumer to process %d payment messages, processed=%d",
		messagesTotal,
		processed.Load(),
	)

	elapsed := time.Since(startedAt)
	throughput := float64(messagesTotal) / elapsed.Seconds()
	snapshot := collectPerfSnapshot(t, registry, groupID, topic)
	avgPayloadBytes := averagePayloadSize(messages)

	require.Equal(t, messagesTotal, processed.Load())
	require.Equal(t, float64(messagesTotal), snapshot.receivedTotal)
	require.Equal(t, float64(messagesTotal), snapshot.successTotal)
	require.Greater(t, snapshot.avgBatchProcessingMs, 0.0)

	t.Logf(
		"\nDQL consumer payment performance result:\n  sent=%d\n  received=%.0f\n  processed=%.0f\n  batches=%.0f\n  avg_batch_size=%.2f\n  avg_payload_bytes=%.2f\n  elapsed=%s\n  throughput=%.2f msg/s\n  avg_batch_processing=%.6f ms/batch\n  avg_message_processing=%.6f ms/msg\n  avg_message_processing_us=%.3f us/msg",
		messagesTotal,
		snapshot.receivedTotal,
		snapshot.successTotal,
		snapshot.batchesProcessed,
		snapshot.avgBatchSize,
		avgPayloadBytes,
		elapsed,
		throughput,
		snapshot.avgBatchProcessingMs,
		snapshot.avgMessageProcessingMs,
		snapshot.avgMessageProcessingUs,
	)
}

func buildPaymentMessages(t *testing.T, topic string, total int64) []*sarama.ProducerMessage {
	t.Helper()

	messages := make([]*sarama.ProducerMessage, 0, total)
	for i := int64(0); i < total; i++ {
		payload, err := buildPaymentMessage(i).Marshal()
		require.NoError(t, err)

		messages = append(messages, &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(fmt.Sprintf("payment-%d", i)),
			Value: sarama.ByteEncoder(payload),
			Headers: []sarama.RecordHeader{
				{Key: []byte("source"), Value: []byte("payments-performance-test")},
				{Key: []byte("event_type"), Value: []byte("payment.authorized")},
			},
		})
	}

	return messages
}

func buildPaymentMessage(i int64) *paymentMessage {
	baseTime := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	return &paymentMessage{
		PaymentID:      fmt.Sprintf("pay_%06d", i),
		ExternalID:     fmt.Sprintf("ext_%06d", i),
		UserID:         fmt.Sprintf("user_%06d", i%1000),
		MerchantID:     fmt.Sprintf("merchant_%03d", i%50),
		OrderID:        fmt.Sprintf("order_%06d", i),
		AmountMinor:    10000 + (i % 5000),
		FeeMinor:       150 + (i % 25),
		Currency:       "RUB",
		Method:         "CARD",
		Status:         "AUTHORIZED",
		Country:        "RU",
		Provider:       "bank-gateway",
		CreatedAt:      baseTime.Add(time.Duration(i) * time.Second),
		AuthorizedAt:   baseTime.Add(time.Duration(i)*time.Second + 2*time.Second),
		Description:    "Subscription renewal payment for premium business account with extended fraud screening and merchant analytics",
		CustomerEmail:  fmt.Sprintf("customer_%06d@example.com", i),
		CustomerPhone:  fmt.Sprintf("+7999%07d", i%10000000),
		IpAddress:      fmt.Sprintf("10.10.%d.%d", i%255, (i/7)%255),
		IdempotencyKey: fmt.Sprintf("idem_%06d", i),
		CardMask:       "411111******1111",
		DeviceID:       fmt.Sprintf("device_%06d", i%2000),
		SessionID:      fmt.Sprintf("session_%06d", i),
		ReferenceCode:  fmt.Sprintf("ref_%06d", i),
		Metadata: map[string]string{
			"channel":       "mobile-app",
			"campaign":      "spring-retention",
			"loyalty_level": "gold",
			"tenant":        "payments-core",
			"installments":  "false",
		},
		LineItems: []paymentLineItem{
			{SKU: "sku-main", Name: "Premium subscription", Quantity: 1, AmountMinor: 9000, Category: "subscription"},
			{SKU: "sku-tax", Name: "VAT", Quantity: 1, AmountMinor: 1000, Category: "tax"},
		},
		BillingAddress: address{
			Country: "RU", City: "Novosibirsk", PostalCode: "630000", Street: "Lenina", Building: "10A",
		},
		ShippingAddress: address{
			Country: "RU", City: "Novosibirsk", PostalCode: "630000", Street: "Krasny Prospect", Building: "5",
		},
		RiskSignals: riskSignals{
			IsVpn:             i%97 == 0,
			IsProxy:           i%113 == 0,
			ChargebackRate:    0.01 + float64(i%10)/1000,
			VelocityPerHour:   int32(2 + i%20),
			VelocityPerDay:    int32(10 + i%100),
			DeviceTrustScore:  0.90 - float64(i%5)/100,
			EmailTrustScore:   0.95 - float64(i%7)/100,
			PhoneTrustScore:   0.91 - float64(i%6)/100,
			MerchantRiskScore: 0.10 + float64(i%8)/100,
		},
		AdditionalFields: map[string]string{
			"issuer_country": "RU",
			"card_brand":     "VISA",
			"bin":            "411111",
			"mcc":            "5734",
			"recurring":      "true",
			"descriptor":     "PREMIUM SERVICE MONTHLY RECURRING PAYMENT FOR BUSINESS ACCOUNT",
		},
	}
}

func validatePayment(m *paymentMessage) error {
	switch {
	case m.PaymentID == "":
		return fmt.Errorf("payment_id is empty")
	case m.UserID == "":
		return fmt.Errorf("user_id is empty")
	case m.MerchantID == "":
		return fmt.Errorf("merchant_id is empty")
	case m.AmountMinor <= 0:
		return fmt.Errorf("amount must be positive")
	case m.FeeMinor < 0:
		return fmt.Errorf("fee must be non-negative")
	case m.Currency != "RUB":
		return fmt.Errorf("unsupported currency")
	case len(m.LineItems) == 0:
		return fmt.Errorf("line items are empty")
	case m.CustomerEmail == "":
		return fmt.Errorf("customer email is empty")
	}
	return nil
}

func averageLineItemAmount(items []paymentLineItem) float64 {
	if len(items) == 0 {
		return 0
	}

	var total int64
	for _, item := range items {
		total += item.AmountMinor * int64(item.Quantity)
	}

	return float64(total) / float64(len(items))
}

func computeRiskScore(signals riskSignals, feeRate float64, avgLineItemAmount float64) float64 {
	score := signals.ChargebackRate*30 +
		float64(signals.VelocityPerHour)*0.2 +
		float64(signals.VelocityPerDay)*0.05 +
		(1-signals.DeviceTrustScore)*20 +
		(1-signals.EmailTrustScore)*15 +
		(1-signals.PhoneTrustScore)*10 +
		signals.MerchantRiskScore*25 +
		feeRate*100 +
		avgLineItemAmount/100000

	if signals.IsVpn {
		score += 5
	}
	if signals.IsProxy {
		score += 7
	}

	return score
}

func buildPaymentChecksum(m *paymentMessage, normalizedEmail string, normalizedDescription string, riskScore float64) string {
	payload := strings.Join([]string{
		m.PaymentID,
		m.ExternalID,
		m.UserID,
		m.MerchantID,
		m.OrderID,
		fmt.Sprintf("%d", m.AmountMinor),
		fmt.Sprintf("%d", m.FeeMinor),
		normalizedEmail,
		normalizedDescription,
		fmt.Sprintf("%.6f", riskScore),
		m.ReferenceCode,
		m.DeviceID,
		m.SessionID,
	}, "|")

	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func averagePayloadSize(messages []*sarama.ProducerMessage) float64 {
	if len(messages) == 0 {
		return 0
	}

	total := 0
	for _, message := range messages {
		if encoder, ok := message.Value.(sarama.ByteEncoder); ok {
			total += len(encoder)
		}
	}

	return float64(total) / float64(len(messages))
}

func TestConsumerPerformancePaymentsWithDLQBatchSave(t *testing.T) {
	if testing.Short() {
		t.Skip("integration performance test is skipped in short mode")
	}

	env := newKafkaIntegrationEnv(t, 5*time.Minute, "dql-payment-dlq-batch-test")
	topic, groupID := env.newTopicAndGroup("dlq-consumer-payments-mixed")
	env.createTopic(topic)

	registry := prometheus.NewRegistry()
	const messagesTotal int64 = 1200

	var (
		batchCalls atomic.Int64
		mu         sync.Mutex
		savedDLQ   = make([]*DLQMessage, 0, messagesTotal/5)
		batchSizes []int
	)

	expectedDecodeFailed, expectedProcessingFailed := expectedMixedFailureCounts(messagesTotal)
	expectedSuccess := messagesTotal - expectedDecodeFailed - expectedProcessingFailed

	consumer, err := NewTopicConsumerGroupDlq[*paymentMessage](
		&Config{
			Topic:   topic,
			Brokers: env.brokers,
			GroupSettings: GroupSettings{
				GroupID:                 groupID,
				OffsetInitial:           "old",
				RebalancedGroupStrategy: "round-robin",
				IsolationLevel:          "commited",
				ReturnErrors:            true,
			},
			ConsumeSettings: ConsumeProcessConfig{
				PerMessageDebugLog: false,
				LogEveryNMessages:  300,
				BatchEnabled:       true,
				BatchSize:          100,
				BatchTimeout:       200 * time.Millisecond,
			},
		},
		env.logger,
		func(ctx context.Context, object *DLQMessage) error {
			return fmt.Errorf("unexpected single DLQ write for message %s", object.ID)
		},
		func(ctx context.Context, object *paymentMessage, values *MapValues) error {
			return validatePayment(object)
		},
		WithPrometheusRegisterer(registry),
		WithDLQBatchSaver(func(ctx context.Context, objects []*DLQMessage) error {
			mu.Lock()
			defer mu.Unlock()

			batchCalls.Add(1)
			batchSizes = append(batchSizes, len(objects))
			savedDLQ = append(savedDLQ, objects...)
			return nil
		}),
	)
	require.NoError(t, err)

	consumerCancel := env.startConsumer(consumer)
	defer stopConsumerGracefully(t, consumer, consumerCancel)

	producer := env.newProducer()
	messages := buildMixedPaymentMessages(t, topic, messagesTotal)

	startedAt := time.Now()
	sendMessages(t, producer, messages)
	waitForCondition(t, 90*time.Second, func() bool {
		snapshot := collectPerfSnapshot(t, registry, groupID, topic)
		mu.Lock()
		defer mu.Unlock()
		return snapshot.successTotal == float64(expectedSuccess) &&
			snapshot.decodeFailedTotal == float64(expectedDecodeFailed) &&
			snapshot.processingFailedTotal == float64(expectedProcessingFailed) &&
			snapshot.dlqTotal == float64(expectedDecodeFailed+expectedProcessingFailed) &&
			len(savedDLQ) == int(expectedDecodeFailed+expectedProcessingFailed)
	}, "timeout waiting for mixed payment processing and DLQ batch persistence")

	elapsed := time.Since(startedAt)
	throughput := float64(messagesTotal) / elapsed.Seconds()
	snapshot := collectPerfSnapshot(t, registry, groupID, topic)

	require.Equal(t, float64(messagesTotal), snapshot.receivedTotal)
	require.Equal(t, float64(expectedSuccess), snapshot.successTotal)
	require.Equal(t, float64(expectedDecodeFailed), snapshot.decodeFailedTotal)
	require.Equal(t, float64(expectedProcessingFailed), snapshot.processingFailedTotal)
	require.Equal(t, float64(expectedDecodeFailed+expectedProcessingFailed), snapshot.dlqTotal)
	require.Greater(t, batchCalls.Load(), int64(0))

	mu.Lock()
	totalDLQSaved := len(savedDLQ)
	observedBatchCalls := len(batchSizes)
	maxBatchSize := maxIntSlice(batchSizes)
	mu.Unlock()

	t.Logf(
		"\nDQL consumer payment mixed result:\n  sent=%d\n  received=%.0f\n  success=%.0f\n  decode_failed=%.0f\n  processing_failed=%.0f\n  dlq_saved=%.0f\n  dlq_batch_calls=%d\n  max_dlq_batch_size=%d\n  elapsed=%s\n  throughput=%.2f msg/s",
		messagesTotal,
		snapshot.receivedTotal,
		snapshot.successTotal,
		snapshot.decodeFailedTotal,
		snapshot.processingFailedTotal,
		snapshot.dlqTotal,
		observedBatchCalls,
		maxBatchSize,
		elapsed,
		throughput,
	)

	require.Equal(t, int(expectedDecodeFailed+expectedProcessingFailed), totalDLQSaved)
}

func buildMixedPaymentMessages(t *testing.T, topic string, total int64) []*sarama.ProducerMessage {
	t.Helper()

	messages := make([]*sarama.ProducerMessage, 0, total)
	for i := int64(0); i < total; i++ {
		var payload []byte
		var err error

		switch {
		case i%15 == 0:
			payload = []byte(`{"broken_json":`)
		case i%10 == 0:
			message := buildPaymentMessage(i)
			message.Currency = "USD"
			payload, err = message.Marshal()
		default:
			payload, err = buildPaymentMessage(i).Marshal()
		}
		require.NoError(t, err)

		messages = append(messages, &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(fmt.Sprintf("payment-%d", i)),
			Value: sarama.ByteEncoder(payload),
			Headers: []sarama.RecordHeader{
				{Key: []byte("source"), Value: []byte("payments-dlq-batch-test")},
				{Key: []byte("event_type"), Value: []byte("payment.authorized")},
			},
		})
	}

	return messages
}

func expectedMixedFailureCounts(total int64) (decodeFailed int64, processingFailed int64) {
	for i := int64(0); i < total; i++ {
		switch {
		case i%15 == 0:
			decodeFailed++
		case i%10 == 0:
			processingFailed++
		}
	}

	return decodeFailed, processingFailed
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, failureMessage string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal(failureMessage)
}

func maxIntSlice(values []int) int {
	maxValue := 0
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}

	return maxValue
}
