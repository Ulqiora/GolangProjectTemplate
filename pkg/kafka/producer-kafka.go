package kafka

import (
	"github.com/IBM/sarama"
)

type ProducerKafka interface {
	TopicName() string
	SendMessages(message ...*sarama.ProducerMessage) error
	SendMessage(message *sarama.ProducerMessage) error
	Runnable
}
