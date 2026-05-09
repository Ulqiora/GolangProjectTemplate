package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	smarttracing "GolangTemplateProject/pkg/smart-span/tracing"
	transactionmanager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus"
	otelattribute "go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	txManager            transactionmanager.TransactionManager
	userRepo             ports.UserAccountRepository
	passwordRepo         ports.PasswordCredentialRepository
	externalIdentityRepo ports.ExternalIdentityRepository
	outboxRepo           ports.OutboxRepository
	passwordHasher       ports.PasswordHasher
	tokenIssuer          ports.TokenIssuer
	yandexClient         ports.YandexIDClient
	stateTokens          ports.StateTokenManager
	log                  logger.Logger
	metrics              *metrics
}

type Dependencies struct {
	TxManager            transactionmanager.TransactionManager
	UserRepo             ports.UserAccountRepository
	PasswordRepo         ports.PasswordCredentialRepository
	ExternalIdentityRepo ports.ExternalIdentityRepository
	OutboxRepo           ports.OutboxRepository
	PasswordHasher       ports.PasswordHasher
	TokenIssuer          ports.TokenIssuer
	YandexClient         ports.YandexIDClient
	StateTokens          ports.StateTokenManager
	Log                  logger.Logger
	Registerer           prometheus.Registerer
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
		passwordRepo:         deps.PasswordRepo,
		externalIdentityRepo: deps.ExternalIdentityRepo,
		outboxRepo:           deps.OutboxRepo,
		passwordHasher:       deps.PasswordHasher,
		tokenIssuer:          deps.TokenIssuer,
		yandexClient:         deps.YandexClient,
		stateTokens:          deps.StateTokens,
		log:                  baseLogger.WithN("auth_service"),
		metrics:              newMetrics(deps.Registerer),
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	ctx, span := smarttracing.GetDefaultTracer().Start(ctx, "auth.register", trace.WithAttributes(
		otelattribute.String("auth.identifier", normalizeLogin(input.Login)),
		otelattribute.String("auth.provider", string(domain.AuthProviderPassword)),
	))
	defer span.End()

	startedAt := time.Now()
	if err := validateRegisterInput(input); err != nil {
		recordSpanError(span, err)
		s.metrics.observe("register", "validation_error", startedAt)
		return nil, err
	}

	login := normalizeLogin(input.Login)
	email := normalizeEmail(input.Email)
	if err := s.ensureUserIsUnique(ctx, login, email); err != nil {
		recordSpanError(span, err)
		s.metrics.observe("register", "duplicate", startedAt)
		return nil, err
	}

	passwordHash, err := s.passwordHasher.Hash(input.Password)
	if err != nil {
		recordSpanError(span, err)
		s.log.Error("Failed to hash password", attribute.String("error", err.Error()))
		s.metrics.observe("register", "hash_error", startedAt)
		return nil, err
	}

	now := time.Now().UTC()
	account := &domain.UserAccount{
		ID:        domain.NewUUIDv7(),
		Login:     login,
		Email:     email,
		FirstName: strings.TrimSpace(input.FirstName),
		LastName:  strings.TrimSpace(input.LastName),
		Provider:  domain.AuthProviderPassword,
		CreatedAt: now,
		UpdatedAt: now,
	}
	credential := &domain.PasswordCredential{
		UserID:       account.ID,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.userRepo.Create(txCtx, account); err != nil {
			return mapCreateError(err)
		}
		if err := s.passwordRepo.Create(txCtx, credential); err != nil {
			return err
		}
		return s.outboxRepo.Create(txCtx, newUserCreatedOutbox(account))
	}); err != nil {
		recordSpanError(span, err)
		s.metrics.observe("register", "tx_error", startedAt)
		return nil, err
	}

	tokens, err := s.tokenIssuer.IssueTokens(account)
	if err != nil {
		recordSpanError(span, err)
		s.metrics.observe("register", "token_error", startedAt)
		return nil, err
	}

	span.SetAttributes(otelattribute.String("auth.user_id", account.ID.String()))
	span.SetStatus(otelcodes.Ok, "registered")
	s.log.Info("User registered", attribute.String("user_id", account.ID.String()), attribute.String("provider", string(account.Provider)))
	s.metrics.observe("register", "success", startedAt)

	return &AuthResult{
		UserID:       account.ID.String(),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Provider:     string(account.Provider),
		CreatedAt:    account.CreatedAt,
	}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	ctx, span := smarttracing.GetDefaultTracer().Start(ctx, "auth.login", trace.WithAttributes(
		otelattribute.String("auth.identifier", strings.TrimSpace(input.Identifier)),
	))
	defer span.End()

	startedAt := time.Now()
	if err := validateLoginInput(input); err != nil {
		recordSpanError(span, err)
		s.metrics.observe("login", "validation_error", startedAt)
		return nil, err
	}

	account, err := s.findUserByIdentifier(ctx, input.Identifier)
	if err != nil {
		recordSpanError(span, err)
		s.metrics.observe("login", "not_found", startedAt)
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	credential, err := s.passwordRepo.GetByUserID(ctx, account.ID.String())
	if err != nil {
		recordSpanError(span, err)
		s.metrics.observe("login", "credential_error", startedAt)
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := s.passwordHasher.Validate(input.Password, credential.PasswordHash); err != nil {
		recordSpanError(span, err)
		s.metrics.observe("login", "invalid_password", startedAt)
		return nil, ErrInvalidCredentials
	}

	tokens, err := s.tokenIssuer.IssueTokens(account)
	if err != nil {
		recordSpanError(span, err)
		s.metrics.observe("login", "token_error", startedAt)
		return nil, err
	}

	span.SetAttributes(otelattribute.String("auth.user_id", account.ID.String()))
	span.SetStatus(otelcodes.Ok, "authenticated")
	s.metrics.observe("login", "success", startedAt)
	return &AuthResult{
		UserID:       account.ID.String(),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Provider:     string(account.Provider),
		CreatedAt:    account.CreatedAt,
	}, nil
}

func (s *Service) GetYandexAuthURL(ctx context.Context) (string, error) {
	ctx, span := smarttracing.GetDefaultTracer().Start(ctx, "auth.yandex.get_url")
	defer span.End()

	if s.stateTokens == nil || s.yandexClient == nil {
		err := ErrUnsupportedAuthState
		recordSpanError(span, err)
		return "", err
	}

	state, err := s.stateTokens.NewYandexState()
	if err != nil {
		recordSpanError(span, err)
		return "", err
	}
	span.SetStatus(otelcodes.Ok, "authorization_url_created")
	return s.yandexClient.BuildAuthURL(state), nil
}

func (s *Service) CompleteYandexLogin(ctx context.Context, input CompleteYandexInput) (*AuthResult, error) {
	ctx, span := smarttracing.GetDefaultTracer().Start(ctx, "auth.yandex.complete", trace.WithAttributes(
		otelattribute.String("auth.provider", string(domain.AuthProviderYandexID)),
	))
	defer span.End()

	startedAt := time.Now()
	if strings.TrimSpace(input.Code) == "" {
		recordSpanError(span, ErrYandexCodeRequired)
		s.metrics.observe("yandex_login", "validation_error", startedAt)
		return nil, ErrYandexCodeRequired
	}
	if strings.TrimSpace(input.State) == "" {
		recordSpanError(span, ErrYandexStateRequired)
		s.metrics.observe("yandex_login", "validation_error", startedAt)
		return nil, ErrYandexStateRequired
	}
	if err := s.stateTokens.VerifyYandexState(input.State); err != nil {
		recordSpanError(span, err)
		s.metrics.observe("yandex_login", "invalid_state", startedAt)
		return nil, err
	}

	profile, err := s.yandexClient.ExchangeCode(ctx, input.Code)
	if err != nil {
		recordSpanError(span, err)
		s.metrics.observe("yandex_login", "exchange_error", startedAt)
		return nil, err
	}
	if strings.TrimSpace(profile.Subject) == "" || strings.TrimSpace(profile.Email) == "" {
		recordSpanError(span, ErrYandexProfileIncomplete)
		s.metrics.observe("yandex_login", "profile_incomplete", startedAt)
		return nil, ErrYandexProfileIncomplete
	}

	identity, err := s.externalIdentityRepo.GetByProviderSubject(ctx, domain.AuthProviderYandexID, profile.Subject)
	if err == nil {
		account, getErr := s.userRepo.GetByID(ctx, identity.UserID.String())
		if getErr != nil {
			recordSpanError(span, getErr)
			return nil, getErr
		}
		tokens, tokenErr := s.tokenIssuer.IssueTokens(account)
		if tokenErr != nil {
			recordSpanError(span, tokenErr)
			return nil, tokenErr
		}
		span.SetAttributes(otelattribute.String("auth.user_id", account.ID.String()))
		span.SetStatus(otelcodes.Ok, "authenticated_existing_identity")
		s.metrics.observe("yandex_login", "success_existing", startedAt)
		return &AuthResult{
			UserID:       account.ID.String(),
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			Provider:     string(domain.AuthProviderYandexID),
			CreatedAt:    account.CreatedAt,
		}, nil
	}
	if !errors.Is(err, ports.ErrNotFound) {
		recordSpanError(span, err)
		s.metrics.observe("yandex_login", "identity_error", startedAt)
		return nil, err
	}

	account, err := s.registerYandexUser(ctx, profile)
	if err != nil {
		recordSpanError(span, err)
		s.metrics.observe("yandex_login", "registration_error", startedAt)
		return nil, err
	}

	tokens, err := s.tokenIssuer.IssueTokens(account)
	if err != nil {
		recordSpanError(span, err)
		s.metrics.observe("yandex_login", "token_error", startedAt)
		return nil, err
	}

	span.SetAttributes(otelattribute.String("auth.user_id", account.ID.String()))
	span.SetStatus(otelcodes.Ok, "authenticated_new_identity")
	s.metrics.observe("yandex_login", "success_new", startedAt)
	return &AuthResult{
		UserID:       account.ID.String(),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Provider:     string(domain.AuthProviderYandexID),
		CreatedAt:    account.CreatedAt,
	}, nil
}

func (s *Service) registerYandexUser(ctx context.Context, profile *ports.YandexUserProfile) (*domain.UserAccount, error) {
	now := time.Now().UTC()
	account, err := s.userRepo.GetByEmail(ctx, normalizeEmail(profile.Email))
	if err != nil && !errors.Is(err, ports.ErrNotFound) {
		return nil, err
	}
	if account == nil {
		account = &domain.UserAccount{
			ID:        domain.NewUUIDv7(),
			Login:     buildExternalLogin(profile.Login, profile.Subject),
			Email:     normalizeEmail(profile.Email),
			FirstName: strings.TrimSpace(profile.FirstName),
			LastName:  strings.TrimSpace(profile.LastName),
			Provider:  domain.AuthProviderYandexID,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	identity := &domain.ExternalIdentity{
		ID:        domain.NewUUIDv7(),
		UserID:    account.ID,
		Provider:  domain.AuthProviderYandexID,
		Subject:   strings.TrimSpace(profile.Subject),
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		if account.CreatedAt.Equal(now) {
			if err := s.userRepo.Create(txCtx, account); err != nil {
				return mapCreateError(err)
			}
			if err := s.outboxRepo.Create(txCtx, newUserCreatedOutbox(account)); err != nil {
				return err
			}
		}
		return s.externalIdentityRepo.Create(txCtx, identity)
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *Service) ensureUserIsUnique(ctx context.Context, login string, email string) error {
	if _, err := s.userRepo.GetByLogin(ctx, login); err == nil {
		return ErrUserAlreadyExists
	} else if !errors.Is(err, ports.ErrNotFound) {
		return err
	}

	if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		return ErrUserAlreadyExists
	} else if !errors.Is(err, ports.ErrNotFound) {
		return err
	}
	return nil
}

func (s *Service) findUserByIdentifier(ctx context.Context, identifier string) (*domain.UserAccount, error) {
	normalized := strings.TrimSpace(identifier)
	if strings.Contains(normalized, "@") {
		return s.userRepo.GetByEmail(ctx, normalizeEmail(normalized))
	}
	return s.userRepo.GetByLogin(ctx, normalizeLogin(normalized))
}

func validateRegisterInput(input RegisterInput) error {
	switch {
	case strings.TrimSpace(input.Login) == "":
		return ErrLoginRequired
	case strings.TrimSpace(input.Email) == "":
		return ErrEmailRequired
	case strings.TrimSpace(input.Password) == "":
		return ErrPasswordRequired
	case strings.TrimSpace(input.FirstName) == "":
		return ErrFirstNameRequired
	case strings.TrimSpace(input.LastName) == "":
		return ErrLastNameRequired
	default:
		return nil
	}
}

func validateLoginInput(input LoginInput) error {
	switch {
	case strings.TrimSpace(input.Identifier) == "":
		return ErrIdentifierRequired
	case strings.TrimSpace(input.Password) == "":
		return ErrPasswordRequired
	default:
		return nil
	}
}

func newUserCreatedOutbox(account *domain.UserAccount) *domain.OutboxMessage {
	message, _ := domain.NewUserCreatedOutboxMessage(account)
	return message
}

func recordSpanError(span smarttracing.SmartSpan, err error) {
	if span == nil || err == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(otelcodes.Error, err.Error())
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeLogin(login string) string {
	return strings.ToLower(strings.TrimSpace(login))
}

func buildExternalLogin(login string, subject string) string {
	normalized := normalizeLogin(login)
	if normalized != "" {
		return normalized
	}
	return "yandex-" + strings.ToLower(strings.TrimSpace(subject))
}

func mapCreateError(err error) error {
	if isUniqueViolation(err) {
		return ErrUserAlreadyExists
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type metrics struct {
	operations *prometheus.CounterVec
	duration   *prometheus.HistogramVec
}

func newMetrics(registerer prometheus.Registerer) *metrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	ops := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "auth_service",
			Name:      "operations_total",
			Help:      "Total number of auth application operations by method and status.",
		},
		[]string{"method", "status"},
	)
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "auth_service",
			Name:      "operation_duration_seconds",
			Help:      "Duration of auth application operations in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)

	if err := registerer.Register(ops); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.CounterVec); ok {
				ops = existing
			}
		}
	}
	if err := registerer.Register(duration); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.HistogramVec); ok {
				duration = existing
			}
		}
	}

	return &metrics{
		operations: ops,
		duration:   duration,
	}
}

func (m *metrics) observe(method string, status string, startedAt time.Time) {
	m.operations.WithLabelValues(method, status).Inc()
	m.duration.WithLabelValues(method, status).Observe(time.Since(startedAt).Seconds())
}
