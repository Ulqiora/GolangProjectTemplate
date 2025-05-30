package consumer

import "github.com/IBM/sarama"

var (
	offsetMap map[string]int64 = map[string]int64{
		"new": sarama.OffsetNewest,
		"old": sarama.OffsetOldest,
	}
	rebalanceStrategyMap map[string]sarama.BalanceStrategy = map[string]sarama.BalanceStrategy{
		"round-robin": sarama.NewBalanceStrategyRoundRobin(),
		"range":       sarama.NewBalanceStrategyRange(),
		"sticky":      sarama.NewBalanceStrategySticky(),
	}
	isolationLevelMap map[string]sarama.IsolationLevel = map[string]sarama.IsolationLevel{
		"commited":   sarama.ReadCommitted,
		"uncommited": sarama.ReadUncommitted,
	}
)

type Config struct {
	// Topic - one topic name
	Topic string `yaml:"topic"`
	// Brokers - hosts of kafka brokers
	Brokers []string `yaml:"brokers"`
	Network Network  `yaml:"network"`
	// CompressionType must be equal [0,1,2,3,4]
	CompressionType int8          `yaml:"compression_type"`
	GroupSettings   GroupSettings `yaml:"group_settings"`
}

type GroupSettings struct {
	RebalancedGroupStrategy string `yaml:"rebalance_strategy"`
	ReturnErrors            bool   `yaml:"return_errors"`
	OffsetInitial           string `yaml:"offset_initial"`
	IsolationLevel          string `yaml:"isolation_level"`
	GroupID                 string `yaml:"group_id"`
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
