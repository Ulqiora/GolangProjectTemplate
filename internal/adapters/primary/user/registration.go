package user

import (
	"context"
	"net/http"
	"time"
	
	user_v1 "GolangTemplateProject/internal/adapters/primary/generated/user"
	"GolangTemplateProject/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	ID           uuid.UUID `json:"id"`
	Login        string    `json:"login"`
	OtpUrl       string    `json:"otp_url"`
	CreatedAt    time.Time `json:"created_at"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
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
		ID:           uuid.MustParse(registrationRequest.ID.String()),
		Login:        registrationRequest.Login,
		OtpUrl:       result.OtpUrl,
		CreatedAt:    result.CreatedAt,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

func (m *Handlers) Register(ctx context.Context, user *user_v1.RegistrationRequest) (*user_v1.RegistrationResponse, error) {
	result, err := m.usecase.Registration(ctx, domain.RegistrationUserInfo{
		Id:        domain.ID(uuid.MustParse(user.Id)),
		Login:     user.Login,
		Email:     user.Email,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		Password:  user.Password,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &user_v1.RegistrationResponse{
		Id:           uuid.UUID(result.UserId).String(),
		Login:        user.Login,
		OtpUrl:       result.OtpUrl,
		CreatedAt:    timestamppb.New(result.CreatedAt),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
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
