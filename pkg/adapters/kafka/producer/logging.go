package producer

const (
	LogProducerConfigured              = "Configured Kafka producer"
	LogTransactionalProducerConfigured = "Configured Kafka transactional producer"
	LogProducerMessageQueued           = "Kafka message queued to async producer"
	LogProducerMessageSent             = "Kafka message sent"
	LogProducerMessagesSent            = "Kafka messages sent"
	LogProducerClosed                  = "Kafka producer closed"
	LogAsyncLoopStarted                = "Kafka async producer loop started"
	LogAsyncLoopStopped                = "Kafka async producer loop stopped"
	LogAsyncErrorsChannelClosed        = "Kafka async producer errors channel closed"
	LogAsyncSuccessesChannelClosed     = "Kafka async producer successes channel closed"
	LogAsyncProducerReturnedError      = "Kafka async producer returned error"
	LogAsyncProducerDeliveredMessage   = "Kafka async producer delivered message"
	LogTxSendFailedAbortSucceeded      = "Kafka transactional send failed, transaction aborted"
	LogTxCommitFailedAbortSucceeded    = "Kafka transactional commit failed, transaction aborted"
	LogTxBeginSucceeded                = "Kafka transactional producer began transaction"
	LogTxCommitSucceeded               = "Kafka transactional producer committed transaction"
	LogTxAbortSucceeded                = "Kafka transactional producer aborted transaction"
)
