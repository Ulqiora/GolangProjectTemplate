package producer

import "errors"

var (
	ErrBuildSaramaConfig       = errors.New("build kafka sarama config")
	ErrCreateSyncProducer      = errors.New("create kafka sync producer")
	ErrCreateAsyncProducer     = errors.New("create kafka async producer")
	ErrCreateTxProducer        = errors.New("create kafka transactional producer")
	ErrTransactionalIDRequired = errors.New("transactional id is required for transactional producer")
	ErrLoggerIsNil             = errors.New("logger is nil")
	ErrSendMessage             = errors.New("send kafka message")
	ErrSendMessages            = errors.New("send kafka messages")
	ErrBeginTransaction        = errors.New("begin kafka transaction")
	ErrCommitTransaction       = errors.New("commit kafka transaction")
	ErrAbortTransaction        = errors.New("abort kafka transaction")
	ErrAddOffsetsToTransaction = errors.New("add offsets to kafka transaction")
	ErrAddMessageToTransaction = errors.New("add message to kafka transaction")
	ErrSerializeMessagePayload = errors.New("serialize kafka message payload")
	ErrLoadTLSClientKeyPair    = errors.New("load tls client key pair")
	ErrAppendTLSRootCert       = errors.New("append tls root certificate")
)
