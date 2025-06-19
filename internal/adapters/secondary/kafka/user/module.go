package user

import (
	"context"

	"GolangTemplateProject/internal/usecase/outbox"
	"GolangTemplateProject/pkg/adapters/kafka"
)

type Module struct {
	consumer kafka.Consumer
	usecase  outbox.UserUsecase
}

func NewUserListener(consumer kafka.Consumer, usecase outbox.UserUsecase) *Module {
	return &Module{
		consumer: consumer,
		usecase:  usecase,
	}
}

func (m *Module) Listen(ctx context.Context) {
	m.consumer.Run(ctx)
}
