package authorization

import (
	"context"
	"fmt"
	"time"

	"GolangTemplateProject/internal/config"
	models "GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/repository/user"
	user_secrets "GolangTemplateProject/internal/repository/user-secrets"
	"GolangTemplateProject/pkg/cripto/aesgcm"
	"GolangTemplateProject/pkg/cripto/bcrypt"
	"GolangTemplateProject/pkg/jwt"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/otp"
	open_telemetry "GolangTemplateProject/pkg/smart-span/tracing"
	transaction_manager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

type UserUsecase struct {
	psqlRepo       user.UserRepository
	psqlUserSecret user_secrets.UserSecretRepository
	crypter        aesgcm.Crypter
	bcrypt         bcrypt.Hasher
	aesgcm         aesgcm.Crypter
	jwt            *jwt.JWTManager
	txManager      transaction_manager.TransactionManager
	logger         logger.Logger
}

func NewUserUsecase(
	psqlRepo user.UserRepository,
	psqlUserSecret user_secrets.UserSecretRepository,
	txManager transaction_manager.TransactionManager,
	logger logger.Logger,
) *UserUsecase {
	return &UserUsecase{
		psqlRepo:       psqlRepo,
		psqlUserSecret: psqlUserSecret,
		crypter:        aesgcm.NewCrypt(config.Get().Auth.Aesgcm),
		bcrypt:         bcrypt.New(config.Get().Auth.Bcrypt),
		aesgcm:         aesgcm.NewCrypt(config.Get().Auth.Aesgcm),
		jwt: jwt.NewJWTManager(
			config.Get().Auth.JWT.SecretKey,
			time.Duration(config.Get().Auth.JWT.TTL)*time.Hour,
		),
		txManager: txManager,
		logger:    logger.With("UserUsecase"),
	}
}

func (u *UserUsecase) Registration(ctx context.Context, user models.RegistrationUserInfo) (models.RegistrationUserResponse, error) {
	const (
		logname = "UserUsecase.Registration"
	)
	var (
		err error
	)

	ctxSpan, span := open_telemetry.GetDefaultTracer().Start(ctx, logname)
	//defer span.End()
	_ = logger.DefaultLogger().With(logname)
	defer func() {
		if err != nil {
			err = fmt.Errorf("%s: %w", logname, err)
		}
	}()
	hashedPassword, err := u.bcrypt.Hash(user.Password)
	if err != nil {
		return models.RegistrationUserResponse{}, err
	}
	secretKey, url, err := otp.GenerateOTPInfo(
		totp.GenerateOpts{
			Issuer:      user.Email,
			AccountName: user.Login,
		},
	)

	secretKeyEncrypted, nonce, err := u.crypter.Encrypt([]byte(secretKey))
	if err != nil {
		span.RecordError(err)
		return models.RegistrationUserResponse{}, err
	}
	accessToken, refreshToken, err := u.jwtGenerate(ctx, user.Email, uuid.UUID(user.Id))
	if err != nil {
		return models.RegistrationUserResponse{}, err
	}

	var (
		userData    *models.User
		userSecrets *models.UserSecrets
	)
	err = u.txManager.Do(ctxSpan, func(ctxTx context.Context) error {
		timeNow := time.Now().In(time.UTC)
		userData, err = u.psqlRepo.CreateUser(ctxTx, &models.User{
			Id:        user.Id,
			Email:     user.Email,
			LastName:  user.Lastname,
			FirstName: user.Firstname,
			Login:     user.Login,
			CreatedAt: timeNow,
			UpdatedAt: timeNow,
		})
		if err != nil {
			return err
		}
		userSecrets, err = u.psqlUserSecret.Create(ctxTx, &models.UserSecrets{
			UserId:         uuid.UUID(user.Id),
			HashedPassword: hashedPassword,
			OtpSecret:      secretKeyEncrypted,
			OtpNonce:       nonce,
			OtpUrl:         url,
			UpdatedAt:      timeNow,
		})

		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return models.RegistrationUserResponse{}, err
	}
	return models.RegistrationUserResponse{
		UserId:       userData.Id,
		OtpUrl:       userSecrets.OtpUrl,
		CreatedAt:    userData.CreatedAt,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

//func (u *UserUsecase) Login(ctx context.Context) (string, error) {
//	var (
//		err     error
//		logname = "UserUsecase.Registration"
//	)
//
//	ctxSpan, span := open_telemetry.GetDefaultTracer().Start(ctx, "")
//	//defer span.End()
//	_ = logger.DefaultLogger().With(attribute.String("name", logname))
//	defer func() {
//		if err != nil {
//			err = fmt.Errorf("%s: %w", logname, err)
//		}
//	}()
//	u.bcrypt.Validate()
//}

func (u *UserUsecase) jwtGenerate(_ context.Context, email string, user uuid.UUID) (string, string, error) {
	accessToken, err := u.jwt.Generate(user.String(), email)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := u.jwt.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}
