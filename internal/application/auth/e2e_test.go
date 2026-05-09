package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	authv1 "GolangTemplateProject/internal/adapters/primary/generated/auth/v1"
	primaryauth "GolangTemplateProject/internal/adapters/primary/grpc/auth"
	applicationauth "GolangTemplateProject/internal/application/auth"
	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	servergrpc "GolangTemplateProject/pkg/adapters/server_grpc"
	"GolangTemplateProject/pkg/mocks"
	smarttracing "GolangTemplateProject/pkg/smart-span/tracing"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
)

func TestAuthGatewayE2ERegisterLoginAndTracePropagation(t *testing.T) {
	exporter := &e2eSpanExporter{}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)))
	oldProvider := otel.GetTracerProvider()
	oldPropagator := otel.GetTextMapPropagator()
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	smarttracing.SetDefaultTracer(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(oldProvider)
		otel.SetTextMapPropagator(oldPropagator)
		smarttracing.SetDefaultTracer(noop.NewTracerProvider())
	})

	env := startAuthE2EEnv(t)
	defer env.closeFn()

	rootCtx, rootSpan := provider.Tracer("auth-e2e-test").Start(context.Background(), "test.client")
	traceID := rootSpan.SpanContext().TraceID()

	registerPayload := []byte(`{"login":"e2e-user","email":"e2e-user@example.com","password":"StrongPass123!","firstName":"E2E","lastName":"User"}`)
	registerResp := doJSON(t, rootCtx, env.client, http.MethodPost, env.registerURL, registerPayload)
	if registerResp.StatusCode != http.StatusOK {
		t.Fatalf("register status = %d, body = %s", registerResp.StatusCode, registerResp.Body)
	}
	var authResponse struct {
		UserID       string `json:"userId"`
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		Provider     string `json:"provider"`
	}
	if err := json.Unmarshal([]byte(registerResp.Body), &authResponse); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if authResponse.UserID == "" || authResponse.AccessToken == "" || authResponse.RefreshToken == "" {
		t.Fatalf("register response does not contain auth tokens: %+v", authResponse)
	}
	if authResponse.Provider != string(domain.AuthProviderPassword) {
		t.Fatalf("provider = %q, want %q", authResponse.Provider, domain.AuthProviderPassword)
	}

	loginPayload := []byte(`{"identifier":"e2e-user","password":"StrongPass123!"}`)
	loginResp := doJSON(t, rootCtx, env.client, http.MethodPost, env.loginURL, loginPayload)
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginResp.StatusCode, loginResp.Body)
	}
	rootSpan.End()
	requireForceFlush(t, provider)

	spans := exporter.Spans()
	requireTraceContains(t, spans, traceID, []string{
		"test.client",
		"POST /v1/auth/register",
		"/auth.v1.AuthService/Register",
		"auth.register",
		"POST /v1/auth/login",
		"/auth.v1.AuthService/Login",
		"auth.login",
	})
	requireParentChain(t, spans, traceID, "POST /v1/auth/register", "test.client")
	requireParentChain(t, spans, traceID, "/auth.v1.AuthService/Register", "POST /v1/auth/register")
	requireParentChain(t, spans, traceID, "auth.register", "/auth.v1.AuthService/Register")
}

type authE2EEnv struct {
	client      *http.Client
	registerURL string
	loginURL    string
	closeFn     func()
}

func startAuthE2EEnv(tb testing.TB) *authE2EEnv {
	tb.Helper()

	registry := prometheus.NewRegistry()
	testLogger := &mocks.Logger{}
	userRepo := newE2EUserRepo()
	passwordRepo := newE2EPasswordRepo()
	service := applicationauth.NewService(applicationauth.Dependencies{
		TxManager:      &mocks.TransactionManager{},
		UserRepo:       userRepo,
		PasswordRepo:   passwordRepo,
		OutboxRepo:     &e2eOutboxRepo{},
		PasswordHasher: e2eHasher{},
		TokenIssuer:    e2eTokenIssuer{},
		Registerer:     registry,
		Log:            testLogger,
	})

	grpcListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("listen grpc: %v", err)
	}
	grpcPort := grpcListener.Addr().(*net.TCPAddr).Port
	grpcServer := servergrpc.NewGrpcServer(
		servergrpc.ServerConfig{Host: "127.0.0.1", Port: grpcPort},
		grpc.ChainUnaryInterceptor(primaryauth.NewUnaryServerInterceptor(testLogger, registry)),
	)
	grpcServer.SetListener(grpcListener)
	grpcServer.Register(func(server *grpc.Server) {
		authv1.RegisterAuthServiceServer(server, primaryauth.NewAuthServiceServer(service, testLogger))
	})
	if err = grpcServer.Start(); err != nil {
		tb.Fatalf("start grpc server: %v", err)
	}

	proxyPort := reserveAuthE2EPort(tb)
	proxy := servergrpc.NewProxy(servergrpc.ProxyConfig{
		Enabled: true,
		Host:    "127.0.0.1",
		Port:    proxyPort,
	})
	proxy.SetGRPCEndpoint(fmt.Sprintf("127.0.0.1:%d", grpcPort))
	primaryauth.RegisterGateway(proxy)
	if err = proxy.Start(context.Background()); err != nil {
		_ = grpcServer.Close()
		tb.Fatalf("start grpc proxy: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	waitAuthE2EHealth(tb, client, fmt.Sprintf("http://127.0.0.1:%d/healthz", proxyPort))

	return &authE2EEnv{
		client:      client,
		registerURL: fmt.Sprintf("http://127.0.0.1:%d/v1/auth/register", proxyPort),
		loginURL:    fmt.Sprintf("http://127.0.0.1:%d/v1/auth/login", proxyPort),
		closeFn: func() {
			_ = proxy.Close()
			_ = grpcServer.Close()
			_ = testLogger.Sync()
		},
	}
}

type jsonResponse struct {
	StatusCode int
	Body       string
}

func doJSON(tb testing.TB, ctx context.Context, client *http.Client, method string, url string, payload []byte) jsonResponse {
	tb.Helper()

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		tb.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := client.Do(req)
	if err != nil {
		tb.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		tb.Fatalf("read response body: %v", err)
	}
	return jsonResponse{StatusCode: resp.StatusCode, Body: string(body)}
}

func waitAuthE2EHealth(tb testing.TB, client *http.Client, healthURL string) {
	tb.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	tb.Fatalf("auth gateway did not become ready on %s", healthURL)
}

func reserveAuthE2EPort(tb testing.TB) int {
	tb.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("reserve free port: %v", err)
	}
	defer lis.Close()
	return lis.Addr().(*net.TCPAddr).Port
}

type e2eSpanExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func (e *e2eSpanExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *e2eSpanExporter) Shutdown(context.Context) error { return nil }

func (e *e2eSpanExporter) Spans() []sdktrace.ReadOnlySpan {
	e.mu.Lock()
	defer e.mu.Unlock()
	spans := make([]sdktrace.ReadOnlySpan, len(e.spans))
	copy(spans, e.spans)
	return spans
}

func requireForceFlush(tb testing.TB, provider *sdktrace.TracerProvider) {
	tb.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := provider.ForceFlush(ctx); err != nil {
		tb.Fatalf("force flush spans: %v", err)
	}
}

func requireTraceContains(tb testing.TB, spans []sdktrace.ReadOnlySpan, traceID trace.TraceID, names []string) {
	tb.Helper()

	missing := make([]string, 0)
	for _, name := range names {
		if findSpan(spans, traceID, name) == nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		tb.Fatalf("trace %s missing spans %v; got %v", traceID.String(), missing, spanNames(spans, traceID))
	}
}

func requireParentChain(tb testing.TB, spans []sdktrace.ReadOnlySpan, traceID trace.TraceID, childName string, parentName string) {
	tb.Helper()

	child := findSpan(spans, traceID, childName)
	if child == nil {
		tb.Fatalf("child span %q not found", childName)
	}
	parent := findSpan(spans, traceID, parentName)
	if parent == nil {
		tb.Fatalf("parent span %q not found", parentName)
	}
	if child.Parent().SpanID() != parent.SpanContext().SpanID() {
		tb.Fatalf("span %q parent = %s, want parent span %q (%s)", childName, child.Parent().SpanID(), parentName, parent.SpanContext().SpanID())
	}
}

func findSpan(spans []sdktrace.ReadOnlySpan, traceID trace.TraceID, name string) sdktrace.ReadOnlySpan {
	for _, span := range spans {
		if span.SpanContext().TraceID() == traceID && span.Name() == name {
			return span
		}
	}
	return nil
}

func spanNames(spans []sdktrace.ReadOnlySpan, traceID trace.TraceID) []string {
	names := make([]string, 0)
	for _, span := range spans {
		if span.SpanContext().TraceID() == traceID {
			names = append(names, span.Name())
		}
	}
	return names
}

type e2eUserRepo struct {
	mu      sync.Mutex
	byID    map[string]*domain.UserAccount
	byLogin map[string]*domain.UserAccount
	byEmail map[string]*domain.UserAccount
}

func newE2EUserRepo() *e2eUserRepo {
	return &e2eUserRepo{
		byID:    map[string]*domain.UserAccount{},
		byLogin: map[string]*domain.UserAccount{},
		byEmail: map[string]*domain.UserAccount{},
	}
}

func (r *e2eUserRepo) GetByID(_ context.Context, id string) (*domain.UserAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	account, ok := r.byID[id]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return account, nil
}

func (r *e2eUserRepo) GetByLogin(_ context.Context, login string) (*domain.UserAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	account, ok := r.byLogin[strings.ToLower(strings.TrimSpace(login))]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return account, nil
}

func (r *e2eUserRepo) GetByEmail(_ context.Context, email string) (*domain.UserAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	account, ok := r.byEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return account, nil
}

func (r *e2eUserRepo) Create(_ context.Context, account *domain.UserAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[account.ID.String()] = account
	r.byLogin[account.Login] = account
	r.byEmail[account.Email] = account
	return nil
}

type e2ePasswordRepo struct {
	mu       sync.Mutex
	byUserID map[string]*domain.PasswordCredential
}

func newE2EPasswordRepo() *e2ePasswordRepo {
	return &e2ePasswordRepo{byUserID: map[string]*domain.PasswordCredential{}}
}

func (r *e2ePasswordRepo) GetByUserID(_ context.Context, userID string) (*domain.PasswordCredential, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	credential, ok := r.byUserID[userID]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return credential, nil
}

func (r *e2ePasswordRepo) Create(_ context.Context, credential *domain.PasswordCredential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byUserID[credential.UserID.String()] = credential
	return nil
}

type e2eOutboxRepo struct{}

func (r *e2eOutboxRepo) Create(context.Context, *domain.OutboxMessage) error { return nil }

func (r *e2eOutboxRepo) PickPending(context.Context, uint) ([]*domain.OutboxMessage, error) {
	return nil, nil
}

func (r *e2eOutboxRepo) MarkProcessing(context.Context, string) error { return nil }
func (r *e2eOutboxRepo) MarkSent(context.Context, string) error       { return nil }
func (r *e2eOutboxRepo) MarkFailed(context.Context, string, string, time.Time) error {
	return nil
}

type e2eHasher struct{}

func (e2eHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (e2eHasher) Validate(password string, hash string) error {
	if "hashed:"+password != hash {
		return fmt.Errorf("invalid password")
	}
	return nil
}

type e2eTokenIssuer struct{}

func (e2eTokenIssuer) IssueTokens(user *domain.UserAccount) (ports.TokenPair, error) {
	return ports.TokenPair{
		AccessToken:  "access-" + user.ID.String(),
		RefreshToken: "refresh-" + user.ID.String(),
	}, nil
}
