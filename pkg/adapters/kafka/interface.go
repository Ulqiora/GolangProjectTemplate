package kafka

import (
	"context"

	"github.com/IBM/sarama"
)

type ProducerKafka interface {
	TopicName() string
	SendMessages(message ...*sarama.ProducerMessage) error
	SendMessage(message *sarama.ProducerMessage) error
	Runnable
}

type Consumer interface {
	Run(ctx context.Context)
	GroupID() string
	ResumePartitions()
	WaitStoppedSession() <-chan struct{}
}
