package async_producer

import (
	"context"
	"fmt"
	"sync"

	"GolangTemplateProject/pkg/adapters/kafka/producer"
	"github.com/IBM/sarama"
)

type TopicProducer struct {
	topic          string
	producerSamara sarama.AsyncProducer
	logger         sarama.StdLogger
}

func NewTopicProducer(addresses []string, config producer.Config, logger sarama.StdLogger) (*TopicProducer, error) {
	saramaConfig, err := producer.BuildProduceConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error building kafka sarama config: %s", err.Error())
	}
	syncProducer, err := sarama.NewAsyncProducer(addresses, saramaConfig)
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
	for _, msg := range message {
		t.producerSamara.Input() <- msg
	}
	return nil
}

func (t *TopicProducer) TopicName() string {
	return t.topic
}

func (t *TopicProducer) Run(ctx context.Context, group *sync.WaitGroup) error {
	group.Add(1)
	go func() {
		defer group.Done()
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

func (t *TopicProducer) CommitTx() error {
	return t.producerSamara.CommitTxn()
}
func (t *TopicProducer) BeginTx() error {
	return t.producerSamara.BeginTxn()
}
func (t *TopicProducer) AbortTx() error {
	return t.producerSamara.AbortTxn()
}
