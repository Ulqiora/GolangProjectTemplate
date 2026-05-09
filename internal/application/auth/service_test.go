package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/pkg/mocks"
)

func TestServiceRegisterCreatesUserCredentialAndOutbox(t *testing.T) {
	userRepo := newFakeUserRepo()
	passwordRepo := newFakePasswordRepo()
	outboxRepo := newFakeOutboxRepo()

	service := NewService(Dependencies{
		TxManager:      &mocks.TransactionManager{},
		UserRepo:       userRepo,
		PasswordRepo:   passwordRepo,
		OutboxRepo:     outboxRepo,
		PasswordHasher: fakeHasher{},
		TokenIssuer:    fakeTokenIssuer{},
		Log:            nil,
	})

	result, err := service.Register(context.Background(), RegisterInput{
		Login:     "TestUser",
		Email:     "User@example.com",
		Password:  "secret",
		FirstName: "Ivan",
		LastName:  "Petrov",
	})
	if err != nil {
		t.Fatalf("register returned error: %v", err)
	}
	if result == nil || result.Provider != string(domain.AuthProviderPassword) {
		t.Fatalf("unexpected auth result: %#v", result)
	}
	if len(outboxRepo.created) != 1 {
		t.Fatalf("expected one outbox message, got %d", len(outboxRepo.created))
	}
	if strings.Contains(string(outboxRepo.created[0].Payload), "email") {
		t.Fatalf("outbox payload contains personal data: %s", string(outboxRepo.created[0].Payload))
	}

	createdUser, err := userRepo.GetByLogin(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("expected created user: %v", err)
	}
	if createdUser.Email != "user@example.com" {
		t.Fatalf("email was not normalized: %#v", createdUser)
	}
}

func TestServiceLoginValidatesCredentials(t *testing.T) {
	userRepo := newFakeUserRepo()
	passwordRepo := newFakePasswordRepo()
	account := seedUser(userRepo, "tester", "tester@example.com", domain.AuthProviderPassword)
	_ = passwordRepo.Create(context.Background(), &domain.PasswordCredential{
		UserID:       account.ID,
		PasswordHash: "hashed:secret",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	})

	service := NewService(Dependencies{
		TxManager:      &mocks.TransactionManager{},
		UserRepo:       userRepo,
		PasswordRepo:   passwordRepo,
		OutboxRepo:     newFakeOutboxRepo(),
		PasswordHasher: fakeHasher{},
		TokenIssuer:    fakeTokenIssuer{},
		Log:            nil,
	})

	if _, err := service.Login(context.Background(), LoginInput{Identifier: "tester", Password: "secret"}); err != nil {
		t.Fatalf("expected successful login, got %v", err)
	}
	if _, err := service.Login(context.Background(), LoginInput{Identifier: "tester", Password: "bad"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestServiceCompleteYandexLoginCreatesLinkedUser(t *testing.T) {
	userRepo := newFakeUserRepo()
	externalRepo := newFakeExternalIdentityRepo()
	outboxRepo := newFakeOutboxRepo()

	service := NewService(Dependencies{
		TxManager:            &mocks.TransactionManager{},
		UserRepo:             userRepo,
		PasswordRepo:         newFakePasswordRepo(),
		ExternalIdentityRepo: externalRepo,
		OutboxRepo:           outboxRepo,
		PasswordHasher:       fakeHasher{},
		TokenIssuer:          fakeTokenIssuer{},
		YandexClient: fakeYandexClient{
			profile: &ports.YandexUserProfile{
				Subject:   "ya-123",
				Login:     "ya-login",
				Email:     "ya@example.com",
				FirstName: "Yana",
				LastName:  "Dex",
			},
		},
		StateTokens: fakeStateTokens{},
		Log:         nil,
	})

	result, err := service.CompleteYandexLogin(context.Background(), CompleteYandexInput{
		Code:  "code",
		State: "state",
	})
	if err != nil {
		t.Fatalf("complete yandex login returned error: %v", err)
	}
	if result.Provider != string(domain.AuthProviderYandexID) {
		t.Fatalf("unexpected provider: %#v", result)
	}
	if len(outboxRepo.created) != 1 {
		t.Fatalf("expected one outbox message, got %d", len(outboxRepo.created))
	}
	if _, err = externalRepo.GetByProviderSubject(context.Background(), domain.AuthProviderYandexID, "ya-123"); err != nil {
		t.Fatalf("expected external identity to be created: %v", err)
	}
}

func TestServiceCompleteYandexLoginForExistingIdentityDoesNotCreateUser(t *testing.T) {
	userRepo := newFakeUserRepo()
	externalRepo := newFakeExternalIdentityRepo()
	account := seedUser(userRepo, "existing", "existing@example.com", domain.AuthProviderYandexID)
	_ = externalRepo.Create(context.Background(), &domain.ExternalIdentity{
		ID:        domain.NewUUIDv7(),
		UserID:    account.ID,
		Provider:  domain.AuthProviderYandexID,
		Subject:   "ya-123",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})

	service := NewService(Dependencies{
		TxManager:            &mocks.TransactionManager{},
		UserRepo:             userRepo,
		PasswordRepo:         newFakePasswordRepo(),
		ExternalIdentityRepo: externalRepo,
		OutboxRepo:           newFakeOutboxRepo(),
		PasswordHasher:       fakeHasher{},
		TokenIssuer:          fakeTokenIssuer{},
		YandexClient: fakeYandexClient{
			profile: &ports.YandexUserProfile{
				Subject:   "ya-123",
				Login:     "existing",
				Email:     "existing@example.com",
				FirstName: "Ex",
				LastName:  "Isting",
			},
		},
		StateTokens: fakeStateTokens{},
		Log:         nil,
	})

	result, err := service.CompleteYandexLogin(context.Background(), CompleteYandexInput{
		Code:  "code",
		State: "state",
	})
	if err != nil {
		t.Fatalf("complete yandex login returned error: %v", err)
	}
	if result.UserID != account.ID.String() {
		t.Fatalf("expected existing user id, got %#v", result)
	}
	if userRepo.createCalls != 1 {
		t.Fatalf("expected no extra user creations, got %d", userRepo.createCalls)
	}
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
	r.byID[account.ID.String()] = account
	r.byLogin[account.Login] = account
	r.byEmail[account.Email] = account
	return nil
}

type fakePasswordRepo struct {
	byUserID map[string]*domain.PasswordCredential
}

func newFakePasswordRepo() *fakePasswordRepo {
	return &fakePasswordRepo{byUserID: map[string]*domain.PasswordCredential{}}
}

func (r *fakePasswordRepo) GetByUserID(_ context.Context, userID string) (*domain.PasswordCredential, error) {
	credential, ok := r.byUserID[userID]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return credential, nil
}

func (r *fakePasswordRepo) Create(_ context.Context, credential *domain.PasswordCredential) error {
	r.byUserID[credential.UserID.String()] = credential
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

func (r *fakeOutboxRepo) MarkProcessing(context.Context, string) error {
	return nil
}

func (r *fakeOutboxRepo) MarkSent(context.Context, string) error {
	return nil
}

func (r *fakeOutboxRepo) MarkFailed(context.Context, string, string, time.Time) error {
	return nil
}

type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (fakeHasher) Validate(password string, hash string) error {
	if "hashed:"+password != hash {
		return errors.New("invalid password")
	}
	return nil
}

type fakeTokenIssuer struct{}

func (fakeTokenIssuer) IssueTokens(user *domain.UserAccount) (ports.TokenPair, error) {
	return ports.TokenPair{
		AccessToken:  "access-" + user.ID.String(),
		RefreshToken: "refresh-" + user.ID.String(),
	}, nil
}

type fakeYandexClient struct {
	profile *ports.YandexUserProfile
}

func (c fakeYandexClient) BuildAuthURL(state string) string {
	return "https://oauth.yandex.test/auth?state=" + state
}

func (c fakeYandexClient) ExchangeCode(context.Context, string) (*ports.YandexUserProfile, error) {
	return c.profile, nil
}

type fakeStateTokens struct{}

func (fakeStateTokens) NewYandexState() (string, error) { return "state", nil }

func (fakeStateTokens) VerifyYandexState(string) error { return nil }

func seedUser(repo *fakeUserRepo, login string, email string, provider domain.AuthProvider) *domain.UserAccount {
	account := &domain.UserAccount{
		ID:        domain.NewUUIDv7(),
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
