package producer

const (
	LogProducerConfigured              = "Configured Kafka producer"
	LogTransactionalProducerConfigured = "Configured Kafka transactional producer"
	LogAsyncLoopStarted                = "Kafka async producer loop started"
	LogAsyncLoopStopped                = "Kafka async producer loop stopped"
	LogAsyncProducerReturnedError      = "Kafka async producer returned error"
	LogAsyncProducerDeliveredMessage   = "Kafka async producer delivered message"
	LogTxSendFailedAbortSucceeded      = "Kafka transactional send failed, transaction aborted"
	LogTxCommitFailedAbortSucceeded    = "Kafka transactional commit failed, transaction aborted"
)
