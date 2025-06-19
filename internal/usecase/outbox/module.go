package outbox

import (
	"context"
	"encoding/json"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/repository/user"
	"GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/adapters/kafka/consumer"
	"github.com/IBM/sarama"
)

type UserUsecase interface {
	Translate(ctx context.Context, object *domain.User, values *consumer.MapValues) error
}

type Usecase struct {
	repo     user.UserRepository
	producer kafka.ProducerKafka
}

func NewUserUsecase(repo user.UserRepository) *Usecase {
	return &Usecase{
		repo: repo,
	}
}

func (u Usecase) Translate(ctx context.Context) error {
	const limit = 10
	users, err := u.repo.GetSomeoneUsers(ctx, limit)
	if err != nil {
		return err
	}
	var messages []*sarama.ProducerMessage
	for _, selectedUser := range users {
		bytes, err := json.Marshal(selectedUser)
		if err != nil {
			return err
		}
		messages = append(messages, &sarama.ProducerMessage{
			Value: sarama.ByteEncoder(bytes),
		})
	}

	err = u.producer.SendMessages(messages...)
	if err != nil {
		return err
	}
	return nil
}
