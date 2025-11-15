package user_secrets

import (
	"context"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
)

type UserSecretRepository interface {
	Create(ctx context.Context, user *domain.UserSecrets) (*domain.UserSecrets, error)
}

type UserSecretRepositoryImpl struct {
	base ports.BaseRepository[*domain.UserSecrets]
}

func NewUserSecretRepository(base ports.BaseRepository[*domain.UserSecrets]) UserSecretRepository {
	return &UserSecretRepositoryImpl{
		base: base,
	}
}

func (u UserSecretRepositoryImpl) Create(ctx context.Context, secret *domain.UserSecrets) (*domain.UserSecrets, error) {
	return u.base.Create(ctx, secret)
}
