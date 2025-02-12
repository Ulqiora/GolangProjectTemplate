package usecase

import (
	"context"
	"time"

	"GolangTemplateProject/config"
	models "GolangTemplateProject/internal/models/user"
	"GolangTemplateProject/internal/user"
	"GolangTemplateProject/internal/user/repository/dto"
	"GolangTemplateProject/pkg/aesgcm"
	"GolangTemplateProject/pkg/bcrypt"
	"GolangTemplateProject/pkg/email"
	open_telemetry "GolangTemplateProject/pkg/open-telemetry"
	"GolangTemplateProject/pkg/otp"
	"github.com/dgrijalva/jwt-go"
	"github.com/pquerna/otp/totp"
)

type UserUsecase struct {
	psqlRepo     user.PsqlRepository
	emailManager email.Manager
	crypter      aesgcm.Crypter
	cfg          *config.Config
}

func NewUserUsecase(psqlRepo user.PsqlRepository) *UserUsecase {
	return &UserUsecase{
		psqlRepo:     psqlRepo,
		emailManager: email.SMTPEmailManager{},
		crypter:      aesgcm.NewCrypt(config.Get().Auth.Aesgcm),
		cfg:          config.Get(),
	}
}

func (u *UserUsecase) Registration(ctx context.Context, user models.RegistrationUserInfo) (models.RegistrationUserResponse, error) {
	ctx, span := open_telemetry.Tracer.Start(ctx, "UserUsecase.Registration")
	defer span.End()

	hashedPassword, err := bcrypt.GenerateHashedPassword(user.Password)
	if err != nil {
		span.RecordError(err)
		return models.RegistrationUserResponse{}, err
	}
	span.AddEvent("generate hashed password complete")

	secretKey, url, err := otp.GenerateOTPInfo(
		totp.GenerateOpts{
			Issuer:      user.Email,
			AccountName: user.Login,
		},
	)
	span.AddEvent("generate OTP objects completed")

	secretKeyEncrypted, nonce, err := u.crypter.Encrypt([]byte(u.cfg.Auth.Aesgcm.SecretKey), []byte(secretKey))
	if err != nil {
		span.RecordError(err)
		return models.RegistrationUserResponse{}, err
	}
	span.AddEvent("secret key encrypted")

	uuid, err := u.psqlRepo.Registration(ctx, dto.RegistrationUserInfoDTO{
		Login:         user.Login,
		Email:         user.Email,
		Password:      hashedPassword,
		OtpSecret:     secretKeyEncrypted,
		UrlOtpCode:    url,
		OtpCryptNonce: nonce,
	})
	if err != nil {
		span.RecordError(err)
		return models.RegistrationUserResponse{}, err
	}
	span.AddEvent("secret key encrypted")

	if err = u.emailManager.SendRegistrationNotification(&email.Message{
		To:      user.Email,
		Subject: "Registration Notification",
		Body:    "",
	}); err != nil {
		span.RecordError(err)
		return models.RegistrationUserResponse{}, err
	}
	span.AddEvent("notification of registration sended")
	return models.RegistrationUserResponse{
		UserUUID: uuid,
		OtpUrl:   url,
	}, err
}

func (u *UserUsecase) Login(ctx context.Context, email string, password string) (string, error) {
	ctx, span := open_telemetry.Tracer.Start(ctx, "UserUsecase.Login")
	defer span.End()

	userInfo, err := u.psqlRepo.GetUserInfo(ctx, email)
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	if err = bcrypt.ValidatePassword(password, userInfo.HashedPassword); err != nil {
		span.RecordError(err)
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email":   email,
		"user_id": userInfo.Id,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Hour * 1).Unix(),
	})

	tokenWithSecret, err := token.SignedString(config.Get().Auth.JWT.SecretKey)
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	return tokenWithSecret, nil
}
