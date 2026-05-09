package vkusers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	smarttracing "GolangTemplateProject/pkg/smart-span/tracing"
	transactionmanager "GolangTemplateProject/pkg/transaction-manager"
	"go.opentelemetry.io/otel/codes"
)

type Service struct {
	txManager            transactionmanager.TransactionManager
	userRepo             ports.UserAccountRepository
	externalIdentityRepo ports.ExternalIdentityRepository
	outboxRepo           ports.OutboxRepository
	log                  logger.Logger
}

type Dependencies struct {
	TxManager            transactionmanager.TransactionManager
	UserRepo             ports.UserAccountRepository
	ExternalIdentityRepo ports.ExternalIdentityRepository
	OutboxRepo           ports.OutboxRepository
	Log                  logger.Logger
}

func NewService(deps Dependencies) *Service {
	baseLogger := deps.Log
	if baseLogger == nil {
		baseLogger = logger.DefaultLogger()
	}
	if baseLogger == nil {
		fallback, err := logger.NewLogger(logger.EnvStage, nil)
		if err != nil {
			panic(err)
		}
		baseLogger = fallback
	}

	return &Service{
		txManager:            deps.TxManager,
		userRepo:             deps.UserRepo,
		externalIdentityRepo: deps.ExternalIdentityRepo,
		outboxRepo:           deps.OutboxRepo,
		log:                  baseLogger.WithN("vk_users_service"),
	}
}

func (s *Service) Accept(ctx context.Context, event *domain.VKUserEvent) error {
	ctx, span := smarttracing.GetDefaultTracer().Start(ctx, "auth.vk.accept_user")
	defer span.End()

	if event == nil {
		err := errors.New("vk user event is nil")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	vkUserID := strings.TrimSpace(event.VKUserID)
	if vkUserID == "" {
		err := errors.New("vk_user_id is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	var createdUserID string
	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		if identity, err := s.externalIdentityRepo.GetByProviderSubject(txCtx, domain.AuthProviderVK, vkUserID); err == nil {
			createdUserID = identity.UserID.String()
			return nil
		} else if !errors.Is(err, ports.ErrNotFound) {
			return err
		}

		account, created, err := s.resolveAccount(txCtx, vkUserID, event)
		if err != nil {
			return err
		}

		identity := &domain.ExternalIdentity{
			ID:        domain.NewUUIDv7(),
			UserID:    account.ID,
			Provider:  domain.AuthProviderVK,
			Subject:   vkUserID,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err = s.externalIdentityRepo.Create(txCtx, identity); err != nil {
			return err
		}

		if created {
			outboxMessage, outboxErr := domain.NewUserCreatedOutboxMessage(account)
			if outboxErr != nil {
				return outboxErr
			}
			if err = s.outboxRepo.Create(txCtx, outboxMessage); err != nil {
				return err
			}
		}

		createdUserID = account.ID.String()
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "accepted")
	s.log.Info(
		"VK user accepted",
		attribute.String("vk_user_id", vkUserID),
		attribute.String("user_id", createdUserID),
	)
	return nil
}

func (s *Service) resolveAccount(ctx context.Context, vkUserID string, event *domain.VKUserEvent) (*domain.UserAccount, bool, error) {
	email := normalizeEmail(event.Email)
	if email != "" {
		if account, err := s.userRepo.GetByEmail(ctx, email); err == nil {
			return account, false, nil
		} else if !errors.Is(err, ports.ErrNotFound) {
			return nil, false, err
		}
	}

	login, err := s.resolveLogin(ctx, vkUserID, event.Login)
	if err != nil {
		return nil, false, err
	}

	now := time.Now().UTC()
	account := &domain.UserAccount{
		ID:        domain.NewUUIDv7(),
		Login:     login,
		Email:     emailOrPlaceholder(email, vkUserID),
		FirstName: defaultIfBlank(event.FirstName, "VK"),
		LastName:  defaultIfBlank(event.LastName, "User"),
		Provider:  domain.AuthProviderVK,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err = s.userRepo.Create(ctx, account); err != nil {
		return nil, false, fmt.Errorf("create vk user account: %w", err)
	}
	return account, true, nil
}

func (s *Service) resolveLogin(ctx context.Context, vkUserID string, login string) (string, error) {
	baseLogin := normalizeLogin(login)
	if baseLogin == "" {
		baseLogin = "vk_" + normalizeLogin(vkUserID)
	}

	if _, err := s.userRepo.GetByLogin(ctx, baseLogin); errors.Is(err, ports.ErrNotFound) {
		return baseLogin, nil
	} else if err != nil {
		return "", err
	}

	fallback := "vk_" + normalizeLogin(vkUserID)
	if fallback == baseLogin {
		return "", fmt.Errorf("vk login %q already exists", baseLogin)
	}
	if _, err := s.userRepo.GetByLogin(ctx, fallback); errors.Is(err, ports.ErrNotFound) {
		return fallback, nil
	} else if err != nil {
		return "", err
	}
	return "", fmt.Errorf("vk fallback login %q already exists", fallback)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeLogin(login string) string {
	return strings.ToLower(strings.TrimSpace(login))
}

func emailOrPlaceholder(email string, vkUserID string) string {
	if email != "" {
		return email
	}
	return "vk-" + normalizeLogin(vkUserID) + "@vk.local"
}

func defaultIfBlank(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
