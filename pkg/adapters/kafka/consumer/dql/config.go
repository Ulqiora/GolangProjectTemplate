package dql

import (
	"time"

	"github.com/IBM/sarama"
)

var (
	offsetMap = map[string]int64{
		"new": sarama.OffsetNewest,
		"old": sarama.OffsetOldest,
	}
	rebalanceStrategyMap = map[string]sarama.BalanceStrategy{
		"round-robin": sarama.NewBalanceStrategyRoundRobin(),
		"range":       sarama.NewBalanceStrategyRange(),
		"sticky":      sarama.NewBalanceStrategySticky(),
	}
	isolationLevelMap = map[string]sarama.IsolationLevel{
		"commited":   sarama.ReadCommitted,
		"uncommited": sarama.ReadUncommitted,
	}
)

type Config struct {
	// Topic - one topic name
	Topic string `yaml:"topic"`
	// Brokers - hosts of kafka brokers
	Brokers Brokers `yaml:"brokers"`
	Network Network `yaml:"network"`
	// CompressionType must be equal [0,1,2,3,4]
	CompressionType int8                 `yaml:"compression_type"`
	GroupSettings   GroupSettings        `yaml:"group_settings"`
	ConsumeSettings ConsumeProcessConfig `yaml:"consume_settings"`
}

type ConsumeProcessConfig struct {
	// dead letter queue
	DlqSave    bool          `yaml:"save_to_dlq"`
	DlqTimeout time.Duration `yaml:"dlq_timeout"`
	DlqRetries int8          `yaml:"dlq_retries"`
	// Reading
	TimeoutReadingMessage    time.Duration `yaml:"timeout_reading_message"`
	MessageProcessingRetries int8          `yaml:"message_processing_retries"`
	// Batch settings
	BatchEnabled bool          `yaml:"batch_enabled"`
	BatchSize    int           `yaml:"batch_size"`
	BatchTimeout time.Duration `yaml:"batch_timeout"`
	// Observability
	PerMessageDebugLog bool `yaml:"per_message_debug_log"`
	LogEveryNMessages  int  `yaml:"log_every_n_messages"`
}

type GroupSettings struct {
	RebalancedGroupStrategy string `yaml:"rebalance_strategy"`
	ReturnErrors            bool   `yaml:"return_errors"`
	OffsetInitial           string `yaml:"offset_initial"`
	IsolationLevel          string `yaml:"isolation_level"`
	GroupID                 string `yaml:"group_id"`
	GroupInstanceId         string `yaml:"group_instance_id"`
}

type Network struct {
	Sasl SASL `yaml:"sasl"`
	TLS  TLS  `yaml:"tls"`
}

type TLS struct {
	Enabled    bool   `yaml:"enabled"`
	ClientCert string `yaml:"client-cert"`
	ClientKey  string `yaml:"client-key"`
	RootCert   string `yaml:"root-cert"`
}

type SASL struct {
	Enable    bool   `yaml:"enable"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Mechanism string `yaml:"mechanism"`
}
