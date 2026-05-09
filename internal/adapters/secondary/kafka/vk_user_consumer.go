package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"GolangTemplateProject/internal/config"
	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	"github.com/IBM/sarama"
)

type VKUserHandler interface {
	Accept(ctx context.Context, event *domain.VKUserEvent) error
}

type VKUserConsumer struct {
	topic   string
	group   sarama.ConsumerGroup
	log     logger.Logger
	handler sarama.ConsumerGroupHandler
}

func NewVKUserConsumer(cfg config.KafkaConsumerConfig, log logger.Logger, handler VKUserHandler) (*VKUserConsumer, error) {
	if !cfg.Enabled {
		return nil, errors.New("vk user consumer is disabled")
	}
	if strings.TrimSpace(cfg.Topic) == "" {
		return nil, errors.New("vk user consumer topic is required")
	}
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("vk user consumer brokers are required")
	}
	if strings.TrimSpace(cfg.GroupID) == "" {
		return nil, errors.New("vk user consumer group_id is required")
	}
	if handler == nil {
		return nil, errors.New("vk user consumer handler is required")
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_3_0_0
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	if cfg.OffsetInitial == "old" {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	}
	saramaConfig.Consumer.IsolationLevel = sarama.ReadCommitted

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("create vk user consumer group: %w", err)
	}

	baseLogger := log
	if baseLogger == nil {
		baseLogger = logger.DefaultLogger()
	}
	if baseLogger == nil {
		return nil, errors.New("logger is nil")
	}

	consumerLogger := baseLogger.With(
		attribute.String("topic", cfg.Topic),
		attribute.String("group_id", cfg.GroupID),
	)

	return &VKUserConsumer{
		topic: cfg.Topic,
		group: group,
		log:   consumerLogger.WithN("vk_user_consumer"),
		handler: &vkUserConsumerGroupHandler{
			handler: handler,
			log:     consumerLogger.WithN("vk_user_consumer_handler"),
		},
	}, nil
}

func (c *VKUserConsumer) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		if err := c.group.Consume(ctx, []string{c.topic}, c.handler); err != nil {
			if ctx.Err() != nil || errors.Is(err, sarama.ErrClosedConsumerGroup) {
				return ctx.Err()
			}
			c.log.Error("VK user consumer group failed", attribute.String("error", err.Error()))
			time.Sleep(time.Second)
		}
	}
	return ctx.Err()
}

func (c *VKUserConsumer) Close() error {
	if c == nil || c.group == nil {
		return nil
	}
	return c.group.Close()
}

type vkUserConsumerGroupHandler struct {
	handler VKUserHandler
	log     logger.Logger
}

func (h *vkUserConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *vkUserConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *vkUserConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if err := h.handleMessage(session.Context(), message); err != nil {
				h.log.Error(
					"VK user message processing failed",
					attribute.String("error", err.Error()),
					attribute.String("topic", message.Topic),
					attribute.Int("partition", int(message.Partition)),
					attribute.Int64("offset", message.Offset),
				)
				continue
			}
			session.MarkMessage(message, "accepted")
		}
	}
}

func (h *vkUserConsumerGroupHandler) handleMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var event domain.VKUserEvent
	if err := event.Unmarshal(message.Value); err != nil {
		return fmt.Errorf("decode vk user event: %w", err)
	}
	return h.handler.Accept(ctx, &event)
}
