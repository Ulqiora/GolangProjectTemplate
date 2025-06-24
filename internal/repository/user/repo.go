package user

import (
	"context"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
)

type UserRepository interface {
	GetUser(ctx context.Context, id domain.ID) (*domain.User, error)
	GetSomeoneUsers(ctx context.Context, limit int) ([]*domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) error
}

type UserRepositoryImpl struct {
	base ports.BaseRepository[domain.User]
}

func NewUserRepository(base ports.BaseRepository[domain.User]) UserRepository {
	return &UserRepositoryImpl{
		base: base,
	}
}

func (u UserRepositoryImpl) GetUser(ctx context.Context, id domain.ID) (*domain.User, error) {
	sql := `select * from users where id = $1;`

	user, err := u.base.SelectOne(ctx, sql, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u UserRepositoryImpl) CreateUser(ctx context.Context, user *domain.User) error {
	return u.base.Create(ctx, user)
}

func (u UserRepositoryImpl) GetSomeoneUsers(ctx context.Context, limit int) ([]*domain.User, error) {
	sql := `SELECT * FROM public.user LIMIT $1;`

	user, err := u.base.Select(ctx, sql, limit)
	if err != nil {
		return nil, err
	}
	return user, nil
}
