package dql

import "errors"

var (
	ErrStartRebalancing         = errors.New("stop consuming from topic, start re-balancing or cluster unavailable")
	ErrMessageUnmarshal         = errors.New("kafka message unmarshal error")
	ErrSaveMessageToDLQDatabase = errors.New("save message to dlq database error")
	ErrMessageProcessingFailed  = errors.New("message processing failed")
	ErrLoadTLSClientKeyPair     = errors.New("load tls client key pair")
	ErrAppendTLSRootCert        = errors.New("append tls root certificate")
	ErrMessageObjectTypeNil     = errors.New("message object type is nil")
	ErrCreateMessageObject      = errors.New("create message object")
)

const (
	MessageProcessingStatusSuccess          = "success"
	MessageProcessingStatusDecodeFailed     = "decode_failed"
	MessageProcessingStatusProcessingFailed = "processing_failed"

	SuccessMessageProcessing = "success message processing"

	LogConsumerLoopStopped                 = "Kafka consumer loop stopped"
	LogConsumerLoopStarted                 = "Starting Kafka DLQ consumer loop"
	LogConsumerStoppedByContext            = "Kafka consumer stopped by context"
	LogConsumerSessionFinished             = "Kafka consumer group session finished, restarting"
	LogConsumerReturnedError               = "Kafka consumer group returned error"
	LogConsumerRestartingAfterError        = "Restarting Kafka consumer group"
	LogConsumerConfigured                  = "Configured Kafka DLQ consumer"
	LogConsumerCreateGroupFailed           = "Failed to create Kafka DLQ consumer group"
	LogSessionStarted                      = "Kafka consumer session started"
	LogSessionSetupOptionFailed            = "Kafka consumer session setup option failed"
	LogSessionCleanupStarted               = "Kafka consumer session cleanup started"
	LogSessionCleanupOptionFailed          = "Kafka consumer session cleanup option failed"
	LogClaimClosed                         = "Kafka consumer claim channel closed, rebalancing"
	LogSessionContextDone                  = "Kafka consumer session context done, committing offsets"
	LogMessageReceived                     = "Kafka message received"
	LogMessageObjectInitializationFailed   = "Failed to initialize message object"
	LogMessageDecodeFailedQueuedForDLQ     = "Kafka message decode failed, message queued for DLQ batch save"
	LogMessageProcessingFailedQueuedForDLQ = "Kafka message processing failed, message queued for DLQ batch save"
	LogMessagesPersistedToDLQSequentially  = "Messages persisted to DLQ storage sequentially"
	LogMessageProcessedSuccessfully        = "Kafka message processed successfully"
	LogDLQBatchPersistedSuccessfully       = "DLQ batch persisted successfully"
	LogDLQBatchPersistFailed               = "Failed to persist DLQ batch"
	LogConsumerProgress                    = "Kafka consumer progress"
	LogConsumerBatchModeEnabled            = "Kafka consumer batch mode enabled"
	LogConsumerProcessingBatch             = "Kafka consumer processing batch"
)
