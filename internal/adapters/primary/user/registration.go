package user

import (
	"net/http"
	"time"

	"GolangTemplateProject/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RegistrationRequest struct {
	ID        uuid.UUID `json:"id"`
	Login     string    `json:"login"`
	Password  string    `json:"password"`
	Firstname string    `json:"firstname"`
	Lastname  string    `json:"lastname"`
	Email     string    `json:"email"`
}

type RegistrationResponse struct {
	ID        uuid.UUID `json:"id"`
	Login     string    `json:"login"`
	OtpUrl    string    `json:"otp_url"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	OtpCode  string `json:"otp_code"`
}

func (m *Handlers) Registration(ctx *gin.Context) {
	var registrationRequest RegistrationRequest
	if err := ctx.ShouldBindJSON(&registrationRequest); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorMessage{Message: err.Error(), Code: http.StatusBadRequest})
		return
	}
	if err := m.validator.Struct(registrationRequest); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorMessage{Message: err.Error(), Code: http.StatusBadRequest})
		return
	}
	result, err := m.usecase.Registration(ctx, domain.RegistrationUserInfo{
		Id:        domain.ID(registrationRequest.ID),
		Login:     registrationRequest.Login,
		Email:     registrationRequest.Email,
		Firstname: registrationRequest.Firstname,
		Lastname:  registrationRequest.Lastname,
		Password:  registrationRequest.Password,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorMessage{Message: err.Error(), Code: http.StatusInternalServerError})
		return
	}
	ctx.JSON(http.StatusOK, RegistrationResponse{
		ID:        uuid.MustParse(registrationRequest.ID.String()),
		Login:     registrationRequest.Login,
		OtpUrl:    result.OtpUrl,
		CreatedAt: result.CreatedAt,
	})
}

func (m *Handlers) Login(ctx *gin.Context) {
	var loginRequest LoginRequest
	if err := ctx.ShouldBindJSON(&loginRequest); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorMessage{Message: err.Error(), Code: http.StatusBadRequest})
		return
	}
	if err := m.validator.Struct(loginRequest); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorMessage{Message: err.Error(), Code: http.StatusBadRequest})
	}

}
