package outbox

import (
	"context"
	"encoding/json"
	"fmt"

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

func NewUserUsecase(repo user.UserRepository, producer kafka.ProducerKafka) *Usecase {
	return &Usecase{
		repo:     repo,
		producer: producer,
	}
}

func (u Usecase) Translate(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		var err error
		//defer func() {
		//	if err != nil {
		//		log.Println("[outbox] translate err:", err)
		//	} else {
		//		log.Println("[outbox] translate success")
		//	}
		//}()
		//log.Println("start to translate")
		const limit = 1
		users, err := u.repo.GetSomeoneUsers(ctx, limit)
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
				Value: sarama.ByteEncoder(bytes),
			})
		}

		err = u.producer.SendMessage(messages[0])
		if err != nil {
			return fmt.Errorf("Usecase/Translate/SendMessage: %w", err)
		}
		return nil
	}
}
