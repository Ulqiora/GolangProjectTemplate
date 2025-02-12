package sync_producer

import (
	"context"

	"GolangTemplateProject/pkg/kafka/producer"
	"github.com/IBM/sarama"
)

type SyncProducer struct {
	topic          string
	producerSamara sarama.SyncProducer
	logger         sarama.StdLogger
}

func NewTopicProducer(addresses []string, config producer.Config) (*SyncProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Errors = config.SaveReturningStatus.Errors
	saramaConfig.Producer.Return.Successes = config.SaveReturningStatus.Succeeded
	saramaConfig.Producer.Compression = sarama.CompressionCodec(config.CompressionType)
	saramaConfig.Producer.RequiredAcks = sarama.RequiredAcks(config.RequiredAcks)

	saramaConfig.Producer.Partitioner = sarama.NewRoundRobinPartitioner

	syncProducer, err := sarama.NewSyncProducer(addresses, saramaConfig)
	if err != nil {
		return nil, err
	}
	return &SyncProducer{
		topic:          config.Topics,
		producerSamara: syncProducer,
	}, nil
}

func (t *SyncProducer) SendMessages(message ...*sarama.ProducerMessage) error {
	return t.producerSamara.SendMessages(message)
}

func (t *SyncProducer) SendMessage(message *sarama.ProducerMessage) error {
	partition, offset, err := t.producerSamara.SendMessage(message)
	if err != nil {
		return err
	}

	t.logger.Printf("Send message to partition %d at offset %d\n", partition, offset)

	return nil
}

func (t *SyncProducer) TopicName() string {
	return t.topic
}

func (t *SyncProducer) Run(_ context.Context) error {
	return nil
}
