package auth

import (
	"context"
	"errors"

	authv1 "GolangTemplateProject/internal/adapters/primary/generated/auth/v1"
	applicationauth "GolangTemplateProject/internal/application/auth"
	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthServiceServer struct {
	authv1.UnimplementedAuthServiceServer
	service *applicationauth.Service
	log     logger.Logger
}

func NewAuthServiceServer(service *applicationauth.Service, log logger.Logger) *AuthServiceServer {
	return &AuthServiceServer{
		service: service,
		log:     log,
	}
}

func (s *AuthServiceServer) Register(ctx context.Context, request *authv1.RegisterRequest) (*authv1.AuthResponse, error) {
	result, err := s.service.Register(ctx, applicationauth.RegisterInput{
		Login:     request.GetLogin(),
		Email:     request.GetEmail(),
		Password:  request.GetPassword(),
		FirstName: request.GetFirstName(),
		LastName:  request.GetLastName(),
	})
	if err != nil {
		return nil, mapAppError(err)
	}
	return toAuthResponse(result), nil
}

func (s *AuthServiceServer) Login(ctx context.Context, request *authv1.LoginRequest) (*authv1.AuthResponse, error) {
	result, err := s.service.Login(ctx, applicationauth.LoginInput{
		Identifier: request.GetIdentifier(),
		Password:   request.GetPassword(),
	})
	if err != nil {
		return nil, mapAppError(err)
	}
	return toAuthResponse(result), nil
}

func (s *AuthServiceServer) GetYandexAuthURL(ctx context.Context, _ *authv1.GetYandexAuthURLRequest) (*authv1.GetYandexAuthURLResponse, error) {
	url, err := s.service.GetYandexAuthURL(ctx)
	if err != nil {
		return nil, mapAppError(err)
	}
	return &authv1.GetYandexAuthURLResponse{AuthorizationUrl: url}, nil
}

func (s *AuthServiceServer) CompleteYandexLogin(ctx context.Context, request *authv1.CompleteYandexLoginRequest) (*authv1.AuthResponse, error) {
	result, err := s.service.CompleteYandexLogin(ctx, applicationauth.CompleteYandexInput{
		Code:  request.GetCode(),
		State: request.GetState(),
	})
	if err != nil {
		return nil, mapAppError(err)
	}
	return toAuthResponse(result), nil
}

func toAuthResponse(result *applicationauth.AuthResult) *authv1.AuthResponse {
	if result == nil {
		return &authv1.AuthResponse{}
	}

	response := &authv1.AuthResponse{
		UserId:       result.UserID,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		Provider:     result.Provider,
	}
	if !result.CreatedAt.IsZero() {
		response.CreatedAt = timestamppb.New(result.CreatedAt)
	}
	return response
}

func mapAppError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, applicationauth.ErrLoginRequired),
		errors.Is(err, applicationauth.ErrEmailRequired),
		errors.Is(err, applicationauth.ErrPasswordRequired),
		errors.Is(err, applicationauth.ErrYandexCodeRequired),
		errors.Is(err, applicationauth.ErrYandexStateRequired),
		errors.Is(err, applicationauth.ErrFirstNameRequired),
		errors.Is(err, applicationauth.ErrLastNameRequired),
		errors.Is(err, applicationauth.ErrIdentifierRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, applicationauth.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, applicationauth.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, applicationauth.ErrUnsupportedAuthState):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, applicationauth.ErrYandexProfileIncomplete):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, ports.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
