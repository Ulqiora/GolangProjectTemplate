//go:build integration
// +build integration

package auth_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	authv1 "GolangTemplateProject/internal/adapters/primary/generated/auth/v1"
	primaryauth "GolangTemplateProject/internal/adapters/primary/grpc/auth"
	applicationauth "GolangTemplateProject/internal/application/auth"
	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	servergrpc "GolangTemplateProject/pkg/adapters/server_grpc"
	"GolangTemplateProject/pkg/mocks"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/grpc"
)

const (
	authLoadTotalRequests = 5000
	authLoadConcurrency   = 100
	authLoadTimeout       = 2 * time.Minute
)

type authLoadEnv struct {
	registry *prometheus.Registry
	loginURL string
	closeFn  func()
}

type authLoadResult struct {
	total        int
	concurrency  int
	elapsed      time.Duration
	successes    int64
	failures     int64
	throughput   float64
	mean         time.Duration
	p50          time.Duration
	p95          time.Duration
	p99          time.Duration
	max          time.Duration
	lastError    string
	statusCounts map[int]int64
}

func TestAuthGatewayLoginLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("integration load test is skipped in short mode")
	}

	env := startAuthLoadEnv(t)
	defer env.closeFn()

	result := runAuthGatewayLoad(t, env.loginURL, authLoadTotalRequests, authLoadConcurrency)
	if result.failures != 0 {
		t.Fatalf("expected zero failed requests, got %d, last error: %s", result.failures, result.lastError)
	}
	if result.successes != int64(authLoadTotalRequests) {
		t.Fatalf("expected %d successful requests, got %d", authLoadTotalRequests, result.successes)
	}
	if result.throughput <= 20 {
		t.Fatalf("throughput is too low: %.2f req/s", result.throughput)
	}

	assertMetricAtLeast(t, env.registry, "auth_grpc_requests_total", map[string]string{
		"method": "/auth.v1.AuthService/Login",
		"code":   "OK",
	}, float64(authLoadTotalRequests))
	assertMetricAtLeast(t, env.registry, "auth_service_operations_total", map[string]string{
		"method": "login",
		"status": "success",
	}, float64(authLoadTotalRequests))

	t.Logf(
		"\nAuth gateway load result:\n  total=%d\n  concurrency=%d\n  successes=%d\n  failures=%d\n  elapsed=%s\n  throughput=%.2f req/s\n  mean=%s\n  p50=%s\n  p95=%s\n  p99=%s\n  max=%s\n  statuses=%v",
		result.total,
		result.concurrency,
		result.successes,
		result.failures,
		result.elapsed,
		result.throughput,
		result.mean,
		result.p50,
		result.p95,
		result.p99,
		result.max,
		result.statusCounts,
	)
}

func BenchmarkAuthGatewayLogin(b *testing.B) {
	env := startAuthLoadEnv(b)
	defer env.closeFn()

	payload := []byte(`{"identifier":"loadtester","password":"StrongPass123!"}`)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        512,
			MaxIdleConnsPerHost: 512,
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest(http.MethodPost, env.loginURL, bytes.NewReader(payload))
			if err != nil {
				b.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("unexpected status code: %d", resp.StatusCode)
			}
		}
	})
}

func startAuthLoadEnv(tb testing.TB) *authLoadEnv {
	tb.Helper()

	registry := prometheus.NewRegistry()
	testLogger := &mocks.Logger{}
	userRepo := newFakeUserRepo()
	passwordRepo := newFakePasswordRepo()
	account := seedUser(userRepo, "loadtester", "loadtester@example.com", domain.AuthProviderPassword)
	_ = passwordRepo.Create(context.Background(), &domain.PasswordCredential{
		UserID:       account.ID,
		PasswordHash: "hashed:StrongPass123!",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	})

	service := applicationauth.NewService(applicationauth.Dependencies{
		TxManager:      &mocks.TransactionManager{},
		UserRepo:       userRepo,
		PasswordRepo:   passwordRepo,
		OutboxRepo:     newFakeOutboxRepo(),
		PasswordHasher: fakeHasher{},
		TokenIssuer:    fakeTokenIssuer{},
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
		authv1.RegisterAuthServiceServer(server, primaryauth.NewAuthServiceServer(service, nil))
	})
	if err = grpcServer.Start(); err != nil {
		tb.Fatalf("start grpc server: %v", err)
	}

	proxyPort := reserveFreePort(tb)
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

	client := &http.Client{Timeout: 2 * time.Second}
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/healthz", proxyPort)
	deadline := time.Now().Add(10 * time.Second)
	for {
		req, reqErr := http.NewRequest(http.MethodGet, healthURL, nil)
		if reqErr == nil {
			resp, doErr := client.Do(req)
			if doErr == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					break
				}
			}
		}
		if time.Now().After(deadline) {
			_ = proxy.Close()
			_ = grpcServer.Close()
			tb.Fatalf("auth gateway did not become ready on %s", healthURL)
		}
		time.Sleep(100 * time.Millisecond)
	}

	return &authLoadEnv{
		registry: registry,
		loginURL: fmt.Sprintf("http://127.0.0.1:%d/v1/auth/login", proxyPort),
		closeFn: func() {
			_ = proxy.Close()
			_ = grpcServer.Close()
			_ = testLogger.Sync()
		},
	}
}

func runAuthGatewayLoad(t *testing.T, loginURL string, total int, concurrency int) authLoadResult {
	t.Helper()

	payload := []byte(`{"identifier":"loadtester","password":"StrongPass123!"}`)
	client := &http.Client{
		Timeout: authLoadTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        concurrency * 2,
			MaxIdleConnsPerHost: concurrency * 2,
		},
	}

	durations := make([]time.Duration, 0, total)
	durationsMu := sync.Mutex{}
	statusCounts := sync.Map{}
	var successes int64
	var failures int64
	var lastError atomic.Value

	jobs := make(chan int)
	startedAt := time.Now()
	var wg sync.WaitGroup
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				requestStartedAt := time.Now()
				req, err := http.NewRequest(http.MethodPost, loginURL, bytes.NewReader(payload))
				if err != nil {
					lastError.Store(err.Error())
					atomic.AddInt64(&failures, 1)
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				elapsed := time.Since(requestStartedAt)
				durationsMu.Lock()
				durations = append(durations, elapsed)
				durationsMu.Unlock()
				if err != nil {
					lastError.Store(err.Error())
					atomic.AddInt64(&failures, 1)
					continue
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				incrementStatusCount(&statusCounts, resp.StatusCode)
				if resp.StatusCode != http.StatusOK {
					lastError.Store(fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
					atomic.AddInt64(&failures, 1)
					continue
				}
				atomic.AddInt64(&successes, 1)
			}
		}()
	}

	for i := 0; i < total; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	elapsed := time.Since(startedAt)
	slices.Sort(durations)

	var totalDuration time.Duration
	for _, duration := range durations {
		totalDuration += duration
	}

	result := authLoadResult{
		total:        total,
		concurrency:  concurrency,
		elapsed:      elapsed,
		successes:    successes,
		failures:     failures,
		throughput:   float64(total) / elapsed.Seconds(),
		mean:         totalDuration / time.Duration(max(len(durations), 1)),
		p50:          percentileDuration(durations, 0.50),
		p95:          percentileDuration(durations, 0.95),
		p99:          percentileDuration(durations, 0.99),
		max:          percentileDuration(durations, 1.0),
		statusCounts: map[int]int64{},
	}

	if value := lastError.Load(); value != nil {
		result.lastError = value.(string)
	}
	statusCounts.Range(func(key any, value any) bool {
		result.statusCounts[key.(int)] = atomic.LoadInt64(value.(*int64))
		return true
	})

	return result
}

func incrementStatusCount(counts *sync.Map, code int) {
	current, _ := counts.LoadOrStore(code, new(int64))
	atomic.AddInt64(current.(*int64), 1)
}

func percentileDuration(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	if percentile <= 0 {
		return values[0]
	}
	index := int(float64(len(values)-1) * percentile)
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func reserveFreePort(tb testing.TB) int {
	tb.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("reserve free port: %v", err)
	}
	defer lis.Close()

	return lis.Addr().(*net.TCPAddr).Port
}

func assertMetricAtLeast(t *testing.T, registry *prometheus.Registry, name string, labels map[string]string, min float64) {
	t.Helper()

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather prometheus metrics: %v", err)
	}

	for _, family := range metricFamilies {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			if !labelsMatch(metric.GetLabel(), labels) {
				continue
			}
			if metric.GetCounter() != nil && metric.GetCounter().GetValue() >= min {
				return
			}
		}
	}

	t.Fatalf("metric %s with labels %v was not >= %.0f", name, labels, min)
}

func labelsMatch(pairs []*dto.LabelPair, expected map[string]string) bool {
	for key, expectedValue := range expected {
		matched := false
		for _, pair := range pairs {
			if pair.GetName() == key && pair.GetValue() == expectedValue {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
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

type fakeOutboxRepo struct{}

func newFakeOutboxRepo() *fakeOutboxRepo { return &fakeOutboxRepo{} }

func (r *fakeOutboxRepo) Create(context.Context, *domain.OutboxMessage) error { return nil }

func (r *fakeOutboxRepo) PickPending(context.Context, uint) ([]*domain.OutboxMessage, error) {
	return nil, nil
}

func (r *fakeOutboxRepo) MarkProcessing(context.Context, string) error { return nil }

func (r *fakeOutboxRepo) MarkSent(context.Context, string) error { return nil }

func (r *fakeOutboxRepo) MarkFailed(context.Context, string, string, time.Time) error { return nil }

type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (fakeHasher) Validate(password string, hash string) error {
	if "hashed:"+password != hash {
		return fmt.Errorf("invalid password")
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

func seedUser(repo *fakeUserRepo, login string, email string, provider domain.AuthProvider) *domain.UserAccount {
	account := &domain.UserAccount{
		ID:        domain.NewUUIDv7(),
		Login:     login,
		Email:     email,
		FirstName: "Load",
		LastName:  "Tester",
		Provider:  provider,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = repo.Create(context.Background(), account)
	return account
}
