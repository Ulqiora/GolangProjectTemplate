package outbox

import (
	"context"
	"fmt"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/repository/user"
	"GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/adapters/kafka/consumer/dql"
	"GolangTemplateProject/pkg/smart-span/tracing"
	"github.com/prometheus/client_golang/prometheus"
)

type UserUsecase interface {
	SaveUser(ctx context.Context, object *domain.User, values *dql.MapValues) error
}

type Usecase struct {
	repo            user.UserRepository
	producer        kafka.TypedProducer[*domain.User]
	countUserSended prometheus.Counter
}

func NewUserUsecase(repo user.UserRepository, producer kafka.TypedProducer[*domain.User]) *Usecase {
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
		if u.producer == nil {
			return fmt.Errorf("Usecase/Translate: producer is nil")
		}

		messages := buildUserMessages(users)
		err = u.producer.SendTypedMessagesTx(messages...)
		if err != nil {
			return fmt.Errorf("Usecase/Translate/SendMessage: %w", err)
		}
		u.countUserSended.Inc()
		return nil
	}
}

func (u Usecase) SaveUser(ctx context.Context, object *domain.User, values *dql.MapValues) error {
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
		if u.producer == nil {
			return fmt.Errorf("Usecase/SaveUser: producer is nil")
		}

		messages := buildUserMessages(users)
		err = u.producer.SendTypedMessagesTx(messages...)
		if err != nil {
			return fmt.Errorf("Usecase/Translate/SendMessage: %w", err)
		}
		u.countUserSended.Inc()
		return nil
	}
}

func buildUserMessages(users []*domain.User) []kafka.TypedMessage[*domain.User] {
	messages := make([]kafka.TypedMessage[*domain.User], 0, len(users))
	for _, selectedUser := range users {
		messages = append(messages, kafka.TypedMessage[*domain.User]{
			Key:   selectedUser.Id.String(),
			Value: selectedUser,
		})
	}
	return messages
}
