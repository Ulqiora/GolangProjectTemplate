package dql

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type baseTopicConsumerGroup[T BaseDqlConsumerModel] struct {
	logger              logger.Logger
	metrics             *consumerMetrics
	groupID             string
	topic               string
	dlqSave             bool
	dlqRetries          int
	dlqTimeout          time.Duration
	batchEnabled        bool
	batchSize           int
	batchTimeout        time.Duration
	perMessageDebugLog  bool
	logEveryNMessages   uint64
	startedAt           time.Time
	receivedCount       atomic.Uint64
	successCount        atomic.Uint64
	decodeFailedCount   atomic.Uint64
	processingFailCount atomic.Uint64
	dlqWriteCount       atomic.Uint64
	execFunction        ExecDomainFunc[T]
	saveFunction        SaveFunction
	saveBatchFunction   SaveBatchFunction
	startSessionOptions []OptionFunc
	stopSessionOptions  []OptionFunc
}

func (b *baseTopicConsumerGroup[T]) Setup(session sarama.ConsumerGroupSession) error {
	fields := append(
		[]attribute.Field{
			attribute.String("group_instance_id", session.MemberID()),
			attribute.Int("generation_id", int(session.GenerationID())),
		},
		claimsAttributes(session.Claims())...,
	)
	b.metrics.activeSessions.WithLabelValues(b.groupID, b.topic).Inc()
	b.metrics.sessionStarts.WithLabelValues(b.groupID, b.topic).Inc()
	b.logger.Info(LogSessionStarted, fields...)
	for _, option := range b.startSessionOptions {
		if err := option(session); err != nil {
			b.logger.Error(
				LogSessionSetupOptionFailed,
				append(fields, attribute.String("error", err.Error()))...,
			)
			return err
		}
	}
	return nil
}

func (b *baseTopicConsumerGroup[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	fields := append(
		[]attribute.Field{
			attribute.String("member_id", session.MemberID()),
			attribute.Int("generation_id", int(session.GenerationID())),
		},
		claimsAttributes(session.Claims())...,
	)
	b.metrics.activeSessions.WithLabelValues(b.groupID, b.topic).Dec()
	b.metrics.sessionStops.WithLabelValues(b.groupID, b.topic, "cleanup").Inc()
	b.logger.Info(LogSessionCleanupStarted, fields...)
	for _, option := range b.stopSessionOptions {
		if err := option(session); err != nil {
			b.logger.Error(
				LogSessionCleanupOptionFailed,
				append(fields, attribute.String("error", err.Error()))...,
			)
			return err
		}
	}
	return nil
}

func (b *baseTopicConsumerGroup[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	mode := b.newClaimMode(claim, func(messages []*sarama.ConsumerMessage, reason string) error {
		return b.processBatch(session, messages, reason)
	})
	defer mode.Stop()

	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				if err := mode.HandleFlush("claim_closed"); err != nil {
					return err
				}
				return b.stopOnClaimClosed(claim)
			}
			if err := mode.HandleMessage(message); err != nil {
				return err
			}
		case <-mode.Timer():
			if err := mode.HandleFlush("batch_timeout"); err != nil {
				return err
			}
		case <-session.Context().Done():
			if err := mode.HandleFlush("context_done"); err != nil {
				return err
			}
			return b.stopOnContextDone(session)
		}
	}
}

func (b *baseTopicConsumerGroup[T]) newClaimMode(
	claim sarama.ConsumerGroupClaim,
	processBatch func(messages []*sarama.ConsumerMessage, reason string) error,
) claimMode {
	if b.batchEnabled && b.batchSize > 1 {
		return newBatchClaimMode(
			b.logger,
			b.metrics,
			b.groupID,
			b.topic,
			b.batchSize,
			b.batchTimeout,
			claim,
			processBatch,
		)
	}

	return newSingleClaimMode(processBatch)
}

func (b *baseTopicConsumerGroup[T]) stopOnClaimClosed(claim sarama.ConsumerGroupClaim) error {
	b.metrics.sessionStops.WithLabelValues(b.groupID, b.topic, "rebalance").Inc()
	b.logger.Info(
		LogClaimClosed,
		attribute.String("topic", claim.Topic()),
		attribute.Int("partition", int(claim.Partition())),
		attribute.Int64("high_watermark", claim.HighWaterMarkOffset()),
	)
	return nil
}

func (b *baseTopicConsumerGroup[T]) stopOnContextDone(session sarama.ConsumerGroupSession) error {
	b.metrics.sessionStops.WithLabelValues(b.groupID, b.topic, "context_done").Inc()
	b.logger.Info(
		LogSessionContextDone,
		attribute.String("topic", b.topic),
		attribute.String("group_id", b.groupID),
	)
	session.Commit()
	return nil
}

func (b *baseTopicConsumerGroup[T]) processBatch(session sarama.ConsumerGroupSession, messages []*sarama.ConsumerMessage, reason string) error {
	ctx := session.Context()
	results := make([]messageProcessingResult, 0, len(messages))
	failedResults := make([]messageProcessingResult, 0, len(messages))

	for _, message := range messages {
		result, err := b.processMessage(ctx, message)
		if err != nil {
			return err
		}
		results = append(results, result)
		if result.dlqMessage != nil {
			failedResults = append(failedResults, result)
		}
	}

	if err := b.persistFailedBatch(ctx, failedResults, reason); err != nil {
		return err
	}

	blockedPartitions := make(map[int32]struct{})
	for _, result := range results {
		b.finalizeMessage(session, result, blockedPartitions)
	}

	return nil
}

func (b *baseTopicConsumerGroup[T]) processMessage(ctx context.Context, message *sarama.ConsumerMessage) (messageProcessingResult, error) {
	startedAt := time.Now()
	attributesMessage := messageAttributes(message)
	b.metrics.messagesReceived.WithLabelValues(b.groupID, b.topic).Inc()
	b.receivedCount.Add(1)
	if b.perMessageDebugLog {
		b.logger.Debug(LogMessageReceived, attributesMessage...)
	}

	object, err := newMessageObject[T]()
	if err != nil {
		b.logger.Error(
			LogMessageObjectInitializationFailed,
			append(attributesMessage, attribute.String("error", err.Error()))...,
		)
		return messageProcessingResult{}, err
	}

	if err = object.Unmarshal(message.Value); err != nil {
		return messageProcessingResult{
			message:       message,
			status:        MessageProcessingStatusDecodeFailed,
			startedAt:     startedAt,
			processingErr: err,
			dlqMessage:    b.newDLQMessage(message, ErrMessageUnmarshal, err),
			markReason:    ErrMessageUnmarshal.Error(),
			logMessage:    LogMessageDecodeFailedQueuedForDLQ,
		}, nil
	}

	err = b.execFunction(ctx, object, NewMapValues(message.Headers))
	if err != nil {
		return messageProcessingResult{
			message:       message,
			status:        MessageProcessingStatusProcessingFailed,
			startedAt:     startedAt,
			processingErr: err,
			dlqMessage:    b.newDLQMessage(message, ErrMessageProcessingFailed, err),
			markReason:    ErrMessageProcessingFailed.Error(),
			logMessage:    LogMessageProcessingFailedQueuedForDLQ,
		}, nil
	}

	return messageProcessingResult{
		message:    message,
		status:     MessageProcessingStatusSuccess,
		startedAt:  startedAt,
		markReason: SuccessMessageProcessing,
	}, nil
}

func (b *baseTopicConsumerGroup[T]) newDLQMessage(message *sarama.ConsumerMessage, err error, description error) *DLQMessage {
	now := time.Now().UTC()
	return &DLQMessage{
		ID:                          uuid.NewString(),
		Topic:                       message.Topic,
		Partition:                   message.Partition,
		Offset:                      message.Offset,
		ObjectIndex:                 0,
		Payload:                     string(message.Value),
		AttemptNumber:               1,
		LastAttemptError:            err.Error(),
		LastAttemptErrorDescription: description.Error(),
		LastAttemptTime:             now,
		Deleted:                     false,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}
}

func (b *baseTopicConsumerGroup[T]) persistFailedBatch(ctx context.Context, failedResults []messageProcessingResult, reason string) error {
	if len(failedResults) == 0 {
		return nil
	}
	if !b.dlqSave {
		b.logger.Warn(
			LogDLQSaveDisabledSkipPersist,
			attribute.String("flush_reason", reason),
			attribute.Int("failed_messages_count", len(failedResults)),
		)
		return nil
	}

	dlqObjects := make([]*DLQMessage, 0, len(failedResults))
	for _, result := range failedResults {
		dlqObjects = append(dlqObjects, result.dlqMessage)
	}

	if b.saveBatchFunction != nil {
		if err := b.runDLQSaveWithRetry(ctx, dlqObjects, func(callCtx context.Context) error {
			return b.saveBatchFunction(callCtx, dlqObjects)
		}); err != nil {
			b.logDLQBatchFailure(failedResults, err, reason)
			return fmt.Errorf("%w: %w", ErrSaveMessageToDLQDatabase, err)
		}
		b.logDLQBatchSuccess(failedResults, reason)
		return nil
	}

	for _, result := range failedResults {
		dlqMessage := result.dlqMessage
		if err := b.runDLQSaveWithRetry(ctx, []*DLQMessage{dlqMessage}, func(callCtx context.Context) error {
			return b.saveFunction(callCtx, dlqMessage)
		}); err != nil {
			b.logDLQBatchFailure(failedResults, err, reason)
			return fmt.Errorf("%w: %w", ErrSaveMessageToDLQDatabase, err)
		}
		b.metrics.dlqWrites.WithLabelValues(b.groupID, b.topic, "success", result.dlqMessage.LastAttemptError).Inc()
		b.dlqWriteCount.Add(1)
	}

	b.logger.Info(
		LogMessagesPersistedToDLQSequentially,
		attribute.String("flush_reason", reason),
		attribute.Int("dlq_messages_count", len(failedResults)),
	)
	return nil
}

func (b *baseTopicConsumerGroup[T]) runDLQSaveWithRetry(
	ctx context.Context,
	objects []*DLQMessage,
	saveFn func(callCtx context.Context) error,
) error {
	attempts := b.dlqRetries
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		now := time.Now().UTC()
		for _, object := range objects {
			object.AttemptNumber = int32(attempt)
			object.LastAttemptTime = now
			object.UpdatedAt = now
		}

		callCtx := ctx
		cancel := func() {}
		if b.dlqTimeout > 0 {
			callCtx, cancel = context.WithTimeout(ctx, b.dlqTimeout)
		}

		err := saveFn(callCtx)
		cancel()
		if err == nil {
			return nil
		}

		lastErr = err
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return lastErr
}

func newBaseDqlConsumer[T BaseDqlConsumerModel](
	logger logger.Logger,
	metrics *consumerMetrics,
	config *Config,
	execFunc ExecDomainFunc[T],
	saveFunc SaveFunction,
	saveBatchFunc SaveBatchFunction,
) *baseTopicConsumerGroup[T] {
	logEveryN := uint64(config.ConsumeSettings.LogEveryNMessages)
	if logEveryN == 0 {
		logEveryN = 1000
	}

	return &baseTopicConsumerGroup[T]{
		logger:             logger.WithN("BaseDqlConsumer"),
		metrics:            metrics,
		groupID:            config.GroupSettings.GroupID,
		topic:              config.Topic,
		dlqSave:            config.ConsumeSettings.DlqSave,
		dlqRetries:         int(config.ConsumeSettings.DlqRetries),
		dlqTimeout:         config.ConsumeSettings.DlqTimeout,
		batchEnabled:       config.ConsumeSettings.BatchEnabled,
		batchSize:          config.ConsumeSettings.BatchSize,
		batchTimeout:       config.ConsumeSettings.BatchTimeout,
		perMessageDebugLog: config.ConsumeSettings.PerMessageDebugLog,
		logEveryNMessages:  logEveryN,
		startedAt:          time.Now(),
		execFunction:       execFunc,
		saveFunction:       saveFunc,
		saveBatchFunction:  saveBatchFunc,
	}
}

func (b *baseTopicConsumerGroup[T]) finalizeMessage(
	session sarama.ConsumerGroupSession,
	result messageProcessingResult,
	blockedPartitions map[int32]struct{},
) {
	attributesMessage := messageAttributes(result.message)
	partition := result.message.Partition
	_, partitionBlocked := blockedPartitions[partition]

	switch result.status {
	case MessageProcessingStatusSuccess:
		b.metrics.messagesProcessed.WithLabelValues(b.groupID, b.topic, "success").Inc()
		b.metrics.processingDuration.WithLabelValues(b.groupID, b.topic, "success").Observe(time.Since(result.startedAt).Seconds())
		b.successCount.Add(1)
		if partitionBlocked {
			b.logger.Warn(
				LogMessageCommitSkippedPartitionBlocked,
				append(attributesMessage, attribute.String("status", MessageProcessingStatusSuccess))...,
			)
		} else {
			session.MarkMessage(result.message, result.markReason)
		}
		if b.perMessageDebugLog {
			b.logger.Debug(
				LogMessageProcessedSuccessfully,
				append(attributesMessage, processingAttributes(result.startedAt, MessageProcessingStatusSuccess)...)...,
			)
		}
		b.logProgressIfNeeded(MessageProcessingStatusSuccess)
	case MessageProcessingStatusDecodeFailed:
		b.metrics.messagesProcessed.WithLabelValues(b.groupID, b.topic, "decode_failed").Inc()
		b.metrics.processingDuration.WithLabelValues(b.groupID, b.topic, "decode_failed").Observe(time.Since(result.startedAt).Seconds())
		b.decodeFailedCount.Add(1)
		b.logger.Warn(
			result.logMessage,
			append(append(attributesMessage, attribute.String("error", result.processingErr.Error())), processingAttributes(result.startedAt, MessageProcessingStatusDecodeFailed)...)...,
		)
		if b.dlqSave && !partitionBlocked {
			session.MarkMessage(result.message, result.markReason)
		} else {
			blockedPartitions[partition] = struct{}{}
			logMessage := LogMessageCommitSkippedPartitionBlocked
			if !b.dlqSave {
				logMessage = LogMessageCommitSkippedDLQDisabled
			}
			b.logger.Warn(
				logMessage,
				append(attributesMessage, attribute.String("status", MessageProcessingStatusDecodeFailed))...,
			)
		}
		b.logProgressIfNeeded(MessageProcessingStatusDecodeFailed)
	case MessageProcessingStatusProcessingFailed:
		b.metrics.messagesProcessed.WithLabelValues(b.groupID, b.topic, "processing_failed").Inc()
		b.metrics.processingDuration.WithLabelValues(b.groupID, b.topic, "processing_failed").Observe(time.Since(result.startedAt).Seconds())
		b.processingFailCount.Add(1)
		b.logger.Error(
			result.logMessage,
			append(append(attributesMessage, attribute.String("error", result.processingErr.Error())), processingAttributes(result.startedAt, MessageProcessingStatusProcessingFailed)...)...,
		)
		if b.dlqSave && !partitionBlocked {
			session.MarkMessage(result.message, result.markReason)
		} else {
			blockedPartitions[partition] = struct{}{}
			logMessage := LogMessageCommitSkippedPartitionBlocked
			if !b.dlqSave {
				logMessage = LogMessageCommitSkippedDLQDisabled
			}
			b.logger.Warn(
				logMessage,
				append(attributesMessage, attribute.String("status", MessageProcessingStatusProcessingFailed))...,
			)
		}
		b.logProgressIfNeeded(MessageProcessingStatusProcessingFailed)
	}
}

func (b *baseTopicConsumerGroup[T]) logDLQBatchSuccess(failedResults []messageProcessingResult, reason string) {
	for _, result := range failedResults {
		b.metrics.dlqWrites.WithLabelValues(b.groupID, b.topic, "success", result.dlqMessage.LastAttemptError).Inc()
		b.dlqWriteCount.Add(1)
	}

	b.logger.Info(
		LogDLQBatchPersistedSuccessfully,
		attribute.String("flush_reason", reason),
		attribute.Int("dlq_messages_count", len(failedResults)),
	)
}

func (b *baseTopicConsumerGroup[T]) logDLQBatchFailure(failedResults []messageProcessingResult, err error, reason string) {
	for _, result := range failedResults {
		b.metrics.dlqWrites.WithLabelValues(b.groupID, b.topic, "failed", result.dlqMessage.LastAttemptError).Inc()
	}

	b.logger.Error(
		LogDLQBatchPersistFailed,
		attribute.String("flush_reason", reason),
		attribute.Int("dlq_messages_count", len(failedResults)),
		attribute.String("error", err.Error()),
	)
}

func (b *baseTopicConsumerGroup[T]) logProgressIfNeeded(lastStatus string) {
	totalProcessed := b.successCount.Load() + b.decodeFailedCount.Load() + b.processingFailCount.Load()
	if totalProcessed == 0 || b.logEveryNMessages == 0 || totalProcessed%b.logEveryNMessages != 0 {
		return
	}

	elapsed := time.Since(b.startedAt)
	throughput := 0.0
	if elapsed > 0 {
		throughput = float64(totalProcessed) / elapsed.Seconds()
	}

	b.logger.Info(
		LogConsumerProgress,
		attribute.String("last_status", lastStatus),
		attribute.Int64("received_total", int64(b.receivedCount.Load())),
		attribute.Int64("processed_total", int64(totalProcessed)),
		attribute.Int64("success_total", int64(b.successCount.Load())),
		attribute.Int64("decode_failed_total", int64(b.decodeFailedCount.Load())),
		attribute.Int64("processing_failed_total", int64(b.processingFailCount.Load())),
		attribute.Int64("dlq_writes_total", int64(b.dlqWriteCount.Load())),
		attribute.Float64("elapsed_seconds", elapsed.Seconds()),
		attribute.Float64("throughput_msg_per_sec", throughput),
	)
}
