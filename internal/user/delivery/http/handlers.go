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

// Login godoc
// @Summary      Login a user
// @Description  Handles user login by processing the provided credentials.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        username body string true "Username"
// @Param        password body string true "Password"
// @Success      200 {object} map[string]bool "Login successful"
// @Failure      500 {object} map[string]bool "Internal server error"
// @Router       /login [post]
func (u UserService) Login() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := u.usecase.Login(c.Context(), "", "")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": false})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": true})
	}
}
