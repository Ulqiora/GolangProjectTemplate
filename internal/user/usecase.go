package user

import (
	"context"

	models "GolangTemplateProject/internal/models/user"
)

type Usecase interface {
	Registration(ctx context.Context, user models.RegistrationUserInfo) (models.RegistrationUserResponse, error)
	Login(ctx context.Context, email string, password string) (string, error)
}
