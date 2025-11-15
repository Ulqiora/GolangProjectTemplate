package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/repository/user"
	"GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/adapters/kafka/consumer"
	"GolangTemplateProject/pkg/smart-span/tracing"
	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

type UserUsecase interface {
	Translate(ctx context.Context, object *domain.User, values *consumer.MapValues) error
}

type Usecase struct {
	repo            user.UserRepository
	producer        kafka.ProducerKafka
	countUserSended prometheus.Counter
}

func NewUserUsecase(repo user.UserRepository, producer kafka.ProducerKafka) *Usecase {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "count_user_sended",
		Help: "count sended object to kafka",
	})
	prometheus.MustRegister(counter)
	return &Usecase{
		repo:            repo,
		producer:        producer,
		countUserSended: counter,
	}
}

func (u Usecase) Translate(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		ctxSpan, span := tracing.GetDefaultTracer().Start(ctx, "")
		defer span.End()
		var err error
		const limit = 1
		users, err := u.repo.GetSomeoneUsers(ctxSpan, limit)
		if err != nil {
			return fmt.Errorf("failed to get users: %w", err)
		}
		var messages []*sarama.ProducerMessage
		for _, selectedUser := range users {
			bytes, err := json.Marshal(selectedUser)
			if err != nil {
				return err
			}
			messages = append(messages, &sarama.ProducerMessage{
				Key:   sarama.StringEncoder(uuid.UUID(selectedUser.Id).String()),
				Value: sarama.ByteEncoder(bytes),
			})
		}

		err = u.producer.SendMessages(messages...)
		if err != nil {
			return fmt.Errorf("Usecase/Translate/SendMessage: %w", err)
		}
		u.countUserSended.Inc()
		return nil
	}
}
