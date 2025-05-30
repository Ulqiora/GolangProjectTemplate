package producer

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/IBM/sarama"
)

type Config struct {
	Env string `yaml:"env"`
	// Topic - one topic name
	Topic string `yaml:"topic"`
	// Brokers - hosts of kafka brokers
	Brokers []string `yaml:"brokers"`
	// CompressionType must be equal [0,1,2,3,4]
	CompressionType int8            `yaml:"compression_type"`
	Network         Network         `yaml:"network"`
	ProduceSettings ProduceSettings `yaml:"produce-settings"`
}

type ProduceSettings struct {
	// RequiredAcks must be equal [-1,0,1]
	RequiredAcks        int16               `yaml:"required_acks"`
	Idempotency         bool                `yaml:"idempotency_key"`
	TransactionalID     string              `yaml:"transactional_id"`
	SaveReturningStatus SaveReturningStatus `yaml:"save_returning_status"`
}

// SaveReturningStatus - settings of saving of messages
type SaveReturningStatus struct {
	// fetch error response in channel
	Errors bool `yaml:"errors"`
	// fetch successes response in channel
	Succeeded bool `yaml:"succeeded"`
}

type Network struct {
	Sasl SASL `yaml:"sasl"`
	TLS  TLS  `yaml:"tls"`
}

type TLS struct {
	Enabled    bool   `yaml:"enabled"`
	ClientCert []byte `yaml:"client-cert"`
	ClientKey  []byte `yaml:"client-key"`
	RootCert   []byte `yaml:"root-cert"`
}

type SASL struct {
	Enable    bool   `yaml:"enable"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Mechanism string `yaml:"mechanism"`
}

func BuildProduceConfig(config Config) (*sarama.Config, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Errors = config.ProduceSettings.SaveReturningStatus.Errors
	saramaConfig.Producer.Return.Successes = config.ProduceSettings.SaveReturningStatus.Succeeded
	saramaConfig.Producer.Idempotent = config.ProduceSettings.Idempotency
	saramaConfig.Producer.Transaction.ID = config.ProduceSettings.TransactionalID
	saramaConfig.Producer.Compression = sarama.CompressionNone
	saramaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	saramaConfig.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	if config.Network.Sasl.Enable == true {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = config.Network.Sasl.Username
		saramaConfig.Net.SASL.Password = config.Network.Sasl.Password
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
	}
	if config.Network.TLS.Enabled {
		saramaConfig.Net.TLS.Enable = true
		cert, err := tls.X509KeyPair(config.Network.TLS.ClientCert, config.Network.TLS.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("error loading client keypair: %v", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(config.Network.TLS.RootCert) {
			return nil, fmt.Errorf("failed to add root certificate")
		}
		saramaConfig.Net.TLS.Config = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		}
	}
	return saramaConfig, nil
}
