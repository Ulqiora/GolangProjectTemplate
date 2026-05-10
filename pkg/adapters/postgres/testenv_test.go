package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
	"github.com/docker/go-connections/nat"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testPostgresImage = "postgres:16-alpine"
	testPostgresPort  = "5432/tcp"
	testPostgresDB    = "postgres"
	testPostgresUser  = "postgres"
	testPostgresPass  = "postgres"
)

type testPostgresEnv struct {
	container tc.Container
	host      string
	port      string
	log       logger.Logger
}

type functionalPostgresEnv struct {
	db   *postgres.Postgres
	host string
	port string
}

func newIntegrationPostgres(t *testing.T, ctx context.Context) *postgres.Postgres {
	t.Helper()

	env := startTestPostgresEnv(t, ctx)
	cfg := &postgres.Config{
		Master:   *integrationEndpoint(env.host, env.port),
		ReadOnly: integrationEndpoint(env.host, env.port),
	}

	return openIntegrationPostgres(t, ctx, cfg, env.host, env.port, env.log)
}

func newIntegrationPostgresWithoutReadOnly(t *testing.T, ctx context.Context) *postgres.Postgres {
	t.Helper()

	env := startTestPostgresEnv(t, ctx)
	cfg := &postgres.Config{
		Master: *integrationEndpoint(env.host, env.port),
	}

	return openIntegrationPostgres(t, ctx, cfg, env.host, env.port, env.log)
}

func newFunctionalPostgres(t *testing.T, ctx context.Context, registry prometheus.Registerer, log logger.Logger) *functionalPostgresEnv {
	t.Helper()

	env := startTestPostgresEnv(t, ctx)
	if log == nil {
		log = env.log
	}

	cfg := &postgres.Config{
		Master:   *integrationEndpoint(env.host, env.port),
		ReadOnly: integrationEndpoint(env.host, env.port),
	}

	db, err := postgres.NewWithOptions(
		ctx,
		cfg,
		postgres.WithLogger(log),
		postgres.WithRegisterer(registry),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	require.NoError(t, db.Ping(ctx))

	return &functionalPostgresEnv{
		db:   db,
		host: env.host,
		port: env.port,
	}
}

func startTestPostgresEnv(t *testing.T, ctx context.Context) *testPostgresEnv {
	t.Helper()

	if testing.Short() {
		t.Skip("postgres integration test is skipped in short mode")
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Skipf("postgres test container is unavailable: %v", recovered)
		}
	}()

	log := newTestLogger(t)
	req := tc.GenericContainerRequest{
		Started: true,
		ContainerRequest: tc.ContainerRequest{
			Image: testPostgresImage,
			Env: map[string]string{
				"POSTGRES_DB":       testPostgresDB,
				"POSTGRES_USER":     testPostgresUser,
				"POSTGRES_PASSWORD": testPostgresPass,
			},
			ExposedPorts: []string{testPostgresPort},
			WaitingFor: wait.ForListeningPort(nat.Port(testPostgresPort)).
				WithStartupTimeout(90 * time.Second),
		},
	}

	container, err := tc.GenericContainer(ctx, req)
	if err != nil {
		t.Skipf("postgres test container is unavailable: %v", err)
	}
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)

	mappedPort, err := container.MappedPort(ctx, nat.Port(testPostgresPort))
	require.NoError(t, err)

	return &testPostgresEnv{
		container: container,
		host:      host,
		port:      mappedPort.Port(),
		log:       log,
	}
}

func openIntegrationPostgres(t *testing.T, ctx context.Context, cfg *postgres.Config, host string, port string, log logger.Logger) *postgres.Postgres {
	t.Helper()

	db, err := postgres.NewWithOptions(ctx, cfg, postgres.WithLogger(log))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	require.NoError(t, db.Ping(ctx), fmt.Sprintf("postgres integration database must be available on %s:%s", host, port))
	return db
}
