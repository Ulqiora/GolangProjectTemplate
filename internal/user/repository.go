package user

import (
	"context"

	models "GolangTemplateProject/internal/models/user"
	"GolangTemplateProject/internal/user/repository/dto"
)

type PsqlRepository interface {
	Registration(ctx context.Context, user dto.RegistrationUserInfoDTO) (models.ID, error)
	GetUserInfo(ctx context.Context, email string) (models.User, error)
}
