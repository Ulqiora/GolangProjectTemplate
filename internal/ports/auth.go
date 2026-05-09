package ports

import (
	"context"
	"time"

	"GolangTemplateProject/internal/domain"
)

type UserAccountRepository interface {
	GetByID(ctx context.Context, id string) (*domain.UserAccount, error)
	GetByLogin(ctx context.Context, login string) (*domain.UserAccount, error)
	GetByEmail(ctx context.Context, email string) (*domain.UserAccount, error)
	Create(ctx context.Context, account *domain.UserAccount) error
}

type PasswordCredentialRepository interface {
	GetByUserID(ctx context.Context, userID string) (*domain.PasswordCredential, error)
	Create(ctx context.Context, credential *domain.PasswordCredential) error
}

type ExternalIdentityRepository interface {
	GetByProviderSubject(ctx context.Context, provider domain.AuthProvider, subject string) (*domain.ExternalIdentity, error)
	Create(ctx context.Context, identity *domain.ExternalIdentity) error
}

type OutboxRepository interface {
	Create(ctx context.Context, message *domain.OutboxMessage) error
	PickPending(ctx context.Context, limit uint) ([]*domain.OutboxMessage, error)
	MarkProcessing(ctx context.Context, id string) error
	MarkSent(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, lastError string, nextAttemptAt time.Time) error
}

type OutboxEventPublisher interface {
	Publish(ctx context.Context, message *domain.OutboxMessage) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Validate(password string, hash string) error
}

type TokenIssuer interface {
	IssueTokens(user *domain.UserAccount) (TokenPair, error)
}

type YandexIDClient interface {
	BuildAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*YandexUserProfile, error)
}

type StateTokenManager interface {
	NewYandexState() (string, error)
	VerifyYandexState(state string) error
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type YandexUserProfile struct {
	Subject   string
	Login     string
	Email     string
	FirstName string
	LastName  string
}
