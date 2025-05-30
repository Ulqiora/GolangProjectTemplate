package sync_producer

import (
	"context"
	"fmt"
	"sync"

	"GolangTemplateProject/pkg/adapters/kafka/producer"
	"github.com/IBM/sarama"
)

type TopicProducer struct {
	topic          string
	producerSamara sarama.SyncProducer
	logger         sarama.StdLogger
}

func NewTopicProducer(addresses []string, config producer.Config, logger sarama.StdLogger) (*TopicProducer, error) {
	saramaConfig, err := producer.BuildProduceConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error building kafka sarama config: %s", err.Error())
	}
	syncProducer, err := sarama.NewSyncProducer(addresses, saramaConfig)
	if err != nil {
		return nil, err
	}
	return &TopicProducer{
		topic:          config.Topic,
		producerSamara: syncProducer,
		logger:         logger,
	}, nil
}

func (t *TopicProducer) SendMessages(message ...*sarama.ProducerMessage) error {
	return t.producerSamara.SendMessages(message)
}

func (t *TopicProducer) SendMessage(message *sarama.ProducerMessage) error {
	partition, offset, err := t.producerSamara.SendMessage(message)
	if err != nil {
		return err
	}

	t.logger.Printf("Send message to partition %d at offset %d\n", partition, offset)

	return nil
}

func (t *TopicProducer) TopicName() string {
	return t.topic
}

func (t *TopicProducer) Run(_ context.Context, _ *sync.WaitGroup) error {
	return nil
}

func (t *TopicProducer) CommitTx() error {
	return t.producerSamara.CommitTxn()
}
func (t *TopicProducer) BeginTx() error {
	return t.producerSamara.BeginTxn()
}
func (t *TopicProducer) AbortTx() error {
	return t.producerSamara.AbortTxn()
}
