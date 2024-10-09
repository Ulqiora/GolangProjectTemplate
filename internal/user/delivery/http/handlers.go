package http

import (
	"time"

	models "GolangTemplateProject/internal/models/user"
	"GolangTemplateProject/internal/user"
	open_telemetry "GolangTemplateProject/pkg/open-telemetry"
	"github.com/gofiber/fiber/v2"
	spanCodes "go.opentelemetry.io/otel/codes"
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

// Register godoc
// @Summary      Register a user
// @Description  Handles user register by processing the provided credentials.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        username {object} user.RegistrationUserInfo true
// @Success      200
// @Failure      500 string true "Internal server error"
// @Router       /registration [post]
func (u UserService) Register() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, span := open_telemetry.Tracer.Start(c.Context(), "UserService.Register")
		defer span.End()
		c.Context()

		time.Sleep(time.Second)

		var info models.RegistrationUserInfo
		if err := c.BodyParser(&info); err != nil {
			span.RecordError(err)
			span.SetStatus(spanCodes.Error, err.Error())
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		userInfoResponse, err := u.usecase.Registration(ctx, info)
		if err != nil {
			span.RecordError(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"status": false})
		}

		span.SetStatus(spanCodes.Ok, "Success Registration")
		return c.Status(fiber.StatusOK).JSON(userInfoResponse)
	}
}

func (u UserService) Login() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, span := open_telemetry.Tracer.Start(c.Context(), "Login")
		defer span.End()
		return nil
	}
}
