package vkusers

import (
	"context"
	"errors"
	"testing"
	"time"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestServiceAcceptCreatesVKUserIdentityAndOutbox(t *testing.T) {
	userRepo := newFakeUserRepo()
	identityRepo := newFakeExternalIdentityRepo()
	outboxRepo := newFakeOutboxRepo()
	service := NewService(Dependencies{
		TxManager:            fakeTxManager{},
		UserRepo:             userRepo,
		ExternalIdentityRepo: identityRepo,
		OutboxRepo:           outboxRepo,
	})

	err := service.Accept(context.Background(), &domain.VKUserEvent{
		VKUserID:  "100500",
		Login:     "VkUser",
		Email:     "VK@example.com",
		FirstName: "Vasya",
		LastName:  "Pupkin",
	})
	require.NoError(t, err)

	account, err := userRepo.GetByEmail(context.Background(), "vk@example.com")
	require.NoError(t, err)
	require.Equal(t, domain.AuthProviderVK, account.Provider)
	require.Equal(t, "vkuser", account.Login)

	identity, err := identityRepo.GetByProviderSubject(context.Background(), domain.AuthProviderVK, "100500")
	require.NoError(t, err)
	require.Equal(t, account.ID, identity.UserID)
	require.Len(t, outboxRepo.created, 1)
	require.Equal(t, domain.UserCreatedTopic, outboxRepo.created[0].Topic)
}

func TestServiceAcceptIsIdempotentWhenVKIdentityExists(t *testing.T) {
	userRepo := newFakeUserRepo()
	identityRepo := newFakeExternalIdentityRepo()
	outboxRepo := newFakeOutboxRepo()
	account := seedUser(userRepo, "existing", "existing@example.com", domain.AuthProviderVK)
	require.NoError(t, identityRepo.Create(context.Background(), &domain.ExternalIdentity{
		ID:        domain.NewUUIDv7(),
		UserID:    account.ID,
		Provider:  domain.AuthProviderVK,
		Subject:   "42",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}))

	service := NewService(Dependencies{
		TxManager:            fakeTxManager{},
		UserRepo:             userRepo,
		ExternalIdentityRepo: identityRepo,
		OutboxRepo:           outboxRepo,
	})

	err := service.Accept(context.Background(), &domain.VKUserEvent{VKUserID: "42"})
	require.NoError(t, err)
	require.Empty(t, outboxRepo.created)
	require.Equal(t, 1, userRepo.createCalls)
}

func TestServiceAcceptLinksExistingAccountByEmail(t *testing.T) {
	userRepo := newFakeUserRepo()
	identityRepo := newFakeExternalIdentityRepo()
	outboxRepo := newFakeOutboxRepo()
	account := seedUser(userRepo, "existing", "same@example.com", domain.AuthProviderPassword)

	service := NewService(Dependencies{
		TxManager:            fakeTxManager{},
		UserRepo:             userRepo,
		ExternalIdentityRepo: identityRepo,
		OutboxRepo:           outboxRepo,
	})

	err := service.Accept(context.Background(), &domain.VKUserEvent{VKUserID: "777", Email: "same@example.com"})
	require.NoError(t, err)

	identity, err := identityRepo.GetByProviderSubject(context.Background(), domain.AuthProviderVK, "777")
	require.NoError(t, err)
	require.Equal(t, account.ID, identity.UserID)
	require.Empty(t, outboxRepo.created)
	require.Equal(t, 1, userRepo.createCalls)
}

type fakeTxManager struct{}

func (fakeTxManager) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (fakeTxManager) Dox(ctx context.Context, fn func(context.Context) error, _ pgx.TxOptions) error {
	return fn(ctx)
}

type fakeUserRepo struct {
	byID        map[string]*domain.UserAccount
	byLogin     map[string]*domain.UserAccount
	byEmail     map[string]*domain.UserAccount
	createCalls int
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:    map[string]*domain.UserAccount{},
		byLogin: map[string]*domain.UserAccount{},
		byEmail: map[string]*domain.UserAccount{},
	}
}

func (r *fakeUserRepo) GetByID(_ context.Context, id string) (*domain.UserAccount, error) {
	account, ok := r.byID[id]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return account, nil
}

func (r *fakeUserRepo) GetByLogin(_ context.Context, login string) (*domain.UserAccount, error) {
	account, ok := r.byLogin[login]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return account, nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.UserAccount, error) {
	account, ok := r.byEmail[email]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return account, nil
}

func (r *fakeUserRepo) Create(_ context.Context, account *domain.UserAccount) error {
	r.createCalls++
	if _, ok := r.byLogin[account.Login]; ok {
		return errors.New("duplicate login")
	}
	if _, ok := r.byEmail[account.Email]; ok {
		return errors.New("duplicate email")
	}
	r.byID[account.ID.String()] = account
	r.byLogin[account.Login] = account
	r.byEmail[account.Email] = account
	return nil
}

type fakeExternalIdentityRepo struct {
	byProviderSubject map[string]*domain.ExternalIdentity
}

func newFakeExternalIdentityRepo() *fakeExternalIdentityRepo {
	return &fakeExternalIdentityRepo{byProviderSubject: map[string]*domain.ExternalIdentity{}}
}

func (r *fakeExternalIdentityRepo) GetByProviderSubject(_ context.Context, provider domain.AuthProvider, subject string) (*domain.ExternalIdentity, error) {
	identity, ok := r.byProviderSubject[string(provider)+":"+subject]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return identity, nil
}

func (r *fakeExternalIdentityRepo) Create(_ context.Context, identity *domain.ExternalIdentity) error {
	r.byProviderSubject[string(identity.Provider)+":"+identity.Subject] = identity
	return nil
}

type fakeOutboxRepo struct {
	created []*domain.OutboxMessage
}

func newFakeOutboxRepo() *fakeOutboxRepo {
	return &fakeOutboxRepo{}
}

func (r *fakeOutboxRepo) Create(_ context.Context, message *domain.OutboxMessage) error {
	r.created = append(r.created, message)
	return nil
}

func (r *fakeOutboxRepo) PickPending(context.Context, uint) ([]*domain.OutboxMessage, error) {
	return nil, nil
}

func (r *fakeOutboxRepo) MarkProcessing(context.Context, string) error { return nil }
func (r *fakeOutboxRepo) MarkSent(context.Context, string) error       { return nil }
func (r *fakeOutboxRepo) MarkFailed(context.Context, string, string, time.Time) error {
	return nil
}

func seedUser(repo *fakeUserRepo, login string, email string, provider domain.AuthProvider) *domain.UserAccount {
	account := &domain.UserAccount{
		ID:        uuid.New(),
		Login:     login,
		Email:     email,
		FirstName: "Test",
		LastName:  "User",
		Provider:  provider,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = repo.Create(context.Background(), account)
	return account
}
