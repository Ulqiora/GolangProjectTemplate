package user

import (
	"GolangTemplateProject/internal/usecase/authorization"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handlers struct {
	router    gin.IRouter
	validator *validator.Validate
	usecase   *authorization.UserUsecase
}

func (m *Handlers) Register(app gin.IRouter) {
	m.router = app
	app.GET("/registration", m.Registration)
	app.GET("/login", m.Login)
}

func NewUserHandlers(usecase *authorization.UserUsecase) *Handlers {
	return &Handlers{
		usecase:   usecase,
		validator: validator.New(),
	}
}

type ErrorMessage struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}
