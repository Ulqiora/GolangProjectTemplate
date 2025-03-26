package postgresql

import (
	"context"
	"errors"

	models "GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/user/repository/dto"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	connection *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{
		connection: db,
	}
}

func (u UserRepository) Registration(ctx context.Context, user dto.RegistrationUserInfoDTO) (models.ID, error) {
	rows, err := u.connection.NamedQueryContext(ctx, queryUserRegister, user)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", errors.New("User already registered")
	}
	var id models.ID
	err = rows.Scan(&id)
	if err != nil {
		return "", errors.New("User registration failed")
	}
	return id, nil
}

func (u UserRepository) GetUserInfo(ctx context.Context, email string) (models.User, error) {
	return models.User{}, nil
}
