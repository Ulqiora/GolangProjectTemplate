package async_producer

import (
	"context"

	"GolangTemplateProject/pkg/adapters/kafka/producer"
	"github.com/IBM/sarama"
)

type TopicProducer struct {
	topic          string
	producerSamara sarama.AsyncProducer
	logger         sarama.StdLogger
}

func NewTopicProducer(addresses []string, config producer.Config, logger sarama.StdLogger) (*TopicProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Errors = config.SaveReturningStatus.Errors
	saramaConfig.Producer.Return.Successes = config.SaveReturningStatus.Succeeded
	saramaConfig.Producer.Compression = sarama.CompressionNone
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	syncProducer, err := sarama.NewAsyncProducer(addresses, saramaConfig)
	if err != nil {
		return nil, err
	}
	return &TopicProducer{
		topic:          config.Topics,
		producerSamara: syncProducer,
		logger:         logger,
	}, nil
}

func (t *TopicProducer) SendMessages(message ...*sarama.ProducerMessage) error {
	for _, msg := range message {
		t.producerSamara.Input() <- msg
	}
	return nil
}

func (t *TopicProducer) TopicName() string {
	return t.topic
}

func (t *TopicProducer) Run(ctx context.Context) error {
	go func() {
		select {
		case err := <-t.producerSamara.Errors():
			t.logger.Println(err)
		case err := <-t.producerSamara.Successes():
			t.logger.Println(err)
		case <-ctx.Done():
			t.logger.Println("End")
			return
		}
	}()
	return nil
}
