package dql

import (
	"time"

	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	"github.com/IBM/sarama"
)

type claimMode interface {
	HandleMessage(message *sarama.ConsumerMessage) error
	HandleFlush(reason string) error
	Timer() <-chan time.Time
	Stop()
}

type singleClaimMode struct {
	processBatch func(messages []*sarama.ConsumerMessage, reason string) error
}

func newSingleClaimMode(processBatch func(messages []*sarama.ConsumerMessage, reason string) error) claimMode {
	return &singleClaimMode{processBatch: processBatch}
}

func (m *singleClaimMode) HandleMessage(message *sarama.ConsumerMessage) error {
	return m.processBatch([]*sarama.ConsumerMessage{message}, "single")
}

func (m *singleClaimMode) HandleFlush(string) error {
	return nil
}

func (m *singleClaimMode) Timer() <-chan time.Time {
	return nil
}

func (m *singleClaimMode) Stop() {}

type batchClaimMode struct {
	logger    logger.Logger
	metrics   *consumerMetrics
	groupID   string
	topic     string
	batchSize int
	timeout   time.Duration
	process   func(messages []*sarama.ConsumerMessage, reason string) error
	messages  []*sarama.ConsumerMessage
	timer     *time.Timer
}

func newBatchClaimMode(
	logger logger.Logger,
	metrics *consumerMetrics,
	groupID string,
	topic string,
	batchSize int,
	timeout time.Duration,
	claim sarama.ConsumerGroupClaim,
	process func(messages []*sarama.ConsumerMessage, reason string) error,
) claimMode {
	logger.Info(
		LogConsumerBatchModeEnabled,
		attribute.Int("batch_size", batchSize),
		attribute.Float64("batch_timeout_ms", float64(timeout.Milliseconds())),
		attribute.String("topic", claim.Topic()),
		attribute.Int("partition", int(claim.Partition())),
	)

	return &batchClaimMode{
		logger:    logger,
		metrics:   metrics,
		groupID:   groupID,
		topic:     topic,
		batchSize: batchSize,
		timeout:   timeout,
		process:   process,
	}
}

func (m *batchClaimMode) HandleMessage(message *sarama.ConsumerMessage) error {
	m.messages = append(m.messages, message)
	if len(m.messages) == 1 {
		m.resetTimer()
	}
	if len(m.messages) < m.batchSize {
		return nil
	}

	return m.HandleFlush("batch_size")
}

func (m *batchClaimMode) HandleFlush(reason string) error {
	if len(m.messages) == 0 {
		return nil
	}
	startedAt := time.Now()

	m.metrics.batchesProcessed.WithLabelValues(m.groupID, m.topic, reason).Inc()
	m.metrics.batchSize.WithLabelValues(m.groupID, m.topic).Observe(float64(len(m.messages)))
	m.logger.Debug(
		LogConsumerProcessingBatch,
		attribute.String("reason", reason),
		attribute.Int("batch_size", len(m.messages)),
		attribute.String("topic", m.topic),
	)

	if err := m.process(m.messages, reason); err != nil {
		return err
	}
	m.metrics.batchDuration.WithLabelValues(m.groupID, m.topic, reason).Observe(time.Since(startedAt).Seconds())

	m.messages = m.messages[:0]
	m.stopTimer()
	return nil
}

func (m *batchClaimMode) Timer() <-chan time.Time {
	if m.timer == nil || len(m.messages) == 0 {
		return nil
	}

	return m.timer.C
}

func (m *batchClaimMode) Stop() {
	m.stopTimer()
}

func (m *batchClaimMode) resetTimer() {
	if m.timeout <= 0 {
		return
	}
	if m.timer == nil {
		m.timer = time.NewTimer(m.timeout)
		return
	}
	if !m.timer.Stop() {
		select {
		case <-m.timer.C:
		default:
		}
	}
	m.timer.Reset(m.timeout)
}

func (m *batchClaimMode) stopTimer() {
	if m.timer == nil {
		return
	}
	if !m.timer.Stop() {
		select {
		case <-m.timer.C:
		default:
		}
	}
}
