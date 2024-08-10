package http

import (
	"GolangTemplateProject/internal/user"
	"github.com/gofiber/fiber/v2"
)

type TracesObject struct {
}

type UserService struct {
	usecase user.Usecase
}

func NewUserService(userUsecase user.Usecase) *UserService {
	return &UserService{
		usecase: userUsecase,
	}
}

// Login
func (u UserService) Login() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := u.usecase.Login(c.Context(), "", "")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.SendStatus(fiber.StatusOK)
	}
}
