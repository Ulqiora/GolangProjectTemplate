package kafka

import (
	"context"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	kafkapkg "GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/adapters/kafka/producer"
	syncproducer "GolangTemplateProject/pkg/adapters/kafka/producer/sync-producer"
	"GolangTemplateProject/pkg/logger"
)

type OutboxPublisher struct {
	producer kafkapkg.ClosableTypedProducer[[]byte]
}

func NewOutboxPublisher(cfg producer.Config, log logger.Logger) (*OutboxPublisher, error) {
	topicProducer, err := syncproducer.NewTopicProducer[[]byte](cfg, log, newOutboxSerializer(cfg))
	if err != nil {
		return nil, err
	}

	return &OutboxPublisher{producer: topicProducer}, nil
}

func (p *OutboxPublisher) Publish(_ context.Context, message *domain.OutboxMessage) error {
	return p.producer.SendTypedMessage(kafkapkg.TypedMessage[[]byte]{
		Topic: message.Topic,
		Key:   message.MessageKey,
		Value: message.Payload,
	})
}

func (p *OutboxPublisher) Close() error {
	return p.producer.Close()
}

var _ ports.OutboxEventPublisher = (*OutboxPublisher)(nil)
