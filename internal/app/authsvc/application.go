package authsvc

import (
	"context"
	"fmt"
	"os"
	"time"

	authv1 "GolangTemplateProject/internal/adapters/primary/generated/auth/v1"
	primaryauth "GolangTemplateProject/internal/adapters/primary/grpc/auth"
	kafkaadapter "GolangTemplateProject/internal/adapters/secondary/kafka"
	"GolangTemplateProject/internal/adapters/secondary/observability"
	"GolangTemplateProject/internal/adapters/secondary/tokens"
	"GolangTemplateProject/internal/adapters/secondary/yandexid"
	applicationauth "GolangTemplateProject/internal/application/auth"
	outboxworker "GolangTemplateProject/internal/application/outbox"
	"GolangTemplateProject/internal/config"
	"GolangTemplateProject/internal/repository/authusers"
	"GolangTemplateProject/internal/repository/externalidentities"
	"GolangTemplateProject/internal/repository/outboxmessages"
	"GolangTemplateProject/internal/repository/passwordcredentials"
	"GolangTemplateProject/pkg/adapters/postgres"
	servergrpc "GolangTemplateProject/pkg/adapters/server_grpc"
	cryptobcrypt "GolangTemplateProject/pkg/cripto/bcrypt"
	"GolangTemplateProject/pkg/logger"
	smarttracing "GolangTemplateProject/pkg/smart-span/tracing"
	transactionmanager "GolangTemplateProject/pkg/transaction-manager"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
)

const DefaultConfigPath = "configs/auth/local.yaml"

type Application struct {
	cfg             *config.ServiceConfig
	log             logger.Logger
	traceShutdown   func(context.Context) error
	postgres        *postgres.Postgres
	outboxPublisher *kafkaadapter.OutboxPublisher
	grpcApp         *servergrpc.App
	metricsServer   *observability.Server
	profilerServer  *observability.Server
	outboxWorker    *outboxworker.Worker
}

func New(configPath string) (*Application, error) {
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	cfg, err := config.LoadServiceConfig(configPath)
	if err != nil {
		return nil, err
	}

	app := &Application{cfg: cfg}
	if err = app.setup(context.Background()); err != nil {
		_ = app.Close(context.Background())
		return nil, err
	}

	return app, nil
}

func (a *Application) Run(ctx context.Context) error {
	if a.metricsServer != nil {
		if err := a.metricsServer.Start(); err != nil {
			return fmt.Errorf("start metrics server: %w", err)
		}
	}
	if a.profilerServer != nil {
		if err := a.profilerServer.Start(); err != nil {
			return fmt.Errorf("start profiler server: %w", err)
		}
	}
	if err := a.grpcApp.Start(ctx); err != nil {
		return err
	}

	go a.outboxWorker.Run(ctx)

	select {
	case <-ctx.Done():
		return a.Close(context.Background())
	case err := <-a.grpcApp.Notify():
		if err != nil {
			_ = a.Close(context.Background())
			return err
		}
		return nil
	}
}

func (a *Application) Close(ctx context.Context) error {
	var closeErr error
	if a.grpcApp != nil {
		if err := a.grpcApp.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	if a.outboxPublisher != nil {
		if err := a.outboxPublisher.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	if a.postgres != nil {
		if err := a.postgres.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	if a.profilerServer != nil {
		if err := a.profilerServer.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	if a.metricsServer != nil {
		if err := a.metricsServer.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	if a.traceShutdown != nil {
		if err := a.traceShutdown(ctx); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	if a.log != nil {
		_ = a.log.Sync()
	}
	return closeErr
}

func (a *Application) setup(ctx context.Context) error {
	if a.cfg == nil {
		return fmt.Errorf("service config is nil")
	}

	traceShutdown := func(context.Context) error { return nil }
	if a.cfg.Tracing.Endpoint != "" {
		_, shutdown, err := smarttracing.InitTracing(ctx, &a.cfg.Tracing)
		if err != nil {
			return fmt.Errorf("init tracing: %w", err)
		}
		traceShutdown = shutdown
	}
	a.traceShutdown = traceShutdown

	appLogger, err := logger.NewLogger(a.cfg.Env, otel.GetTracerProvider().Tracer(a.cfg.ServiceName))
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	logger.SetDefaultLogger(appLogger)
	a.log = appLogger.WithN("auth_app")

	db, err := postgres.NewWithOptions(ctx, &a.cfg.Database.Postgres, postgres.WithLogger(a.log), postgres.WithTracing())
	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}
	a.postgres = db

	txManager := transactionmanager.NewWithOptions(db, transactionmanager.WithLogger(a.log))
	userRepo := authusers.New(db)
	passwordRepo := passwordcredentials.New(db)
	externalIdentityRepo := externalidentities.New(db)
	outboxRepo := outboxmessages.New(db)

	tokenService := tokens.New(
		a.cfg.Auth.JWT.SecretKey,
		time.Duration(a.cfg.Auth.JWT.AccessTTLSeconds)*time.Second,
		time.Duration(a.cfg.Auth.JWT.RefreshTTLSeconds)*time.Second,
	)
	passwordHasher := cryptobcrypt.New(a.cfg.Auth.Bcrypt)
	yandexClient := yandexid.New(a.cfg.AuthService.YandexID)

	authService := applicationauth.NewService(applicationauth.Dependencies{
		TxManager:            txManager,
		UserRepo:             userRepo,
		PasswordRepo:         passwordRepo,
		ExternalIdentityRepo: externalIdentityRepo,
		OutboxRepo:           outboxRepo,
		PasswordHasher:       passwordHasher,
		TokenIssuer:          tokenService,
		YandexClient:         yandexClient,
		StateTokens:          tokenService,
		Log:                  a.log,
	})

	publisher, err := kafkaadapter.NewOutboxPublisher(a.cfg.AuthService.KafkaProducer, a.log)
	if err != nil {
		return fmt.Errorf("init kafka publisher: %w", err)
	}
	a.outboxPublisher = publisher
	a.outboxWorker = outboxworker.NewWorker(txManager, outboxRepo, publisher, a.log, nil)

	grpcApp, err := servergrpc.NewApp(
		a.cfg.AuthService.Server,
		grpc.ChainUnaryInterceptor(primaryauth.NewUnaryServerInterceptor(a.log, nil)),
	)
	if err != nil {
		return fmt.Errorf("init grpc app: %w", err)
	}

	authPrimary := primaryauth.NewAuthServiceServer(authService, a.log)

	grpcApp.GRPC().Register(
		func(server *grpc.Server) {
			authv1.RegisterAuthServiceServer(server, authPrimary)
		},
	)
	primaryauth.RegisterGateway(grpcApp.Proxy())
	a.grpcApp = grpcApp

	if a.cfg.AuthService.Observability.Metrics.Enabled {
		a.metricsServer = observability.NewMetricsServer(a.cfg.AuthService.Observability.Metrics)
	}
	if a.cfg.AuthService.Observability.Profiler.Enabled && a.cfg.Env != logger.EnvProd {
		a.profilerServer = observability.NewProfilerServer(a.cfg.AuthService.Observability.Profiler)
	}

	a.log.Info("Auth service dependencies configured")
	return nil
}

func ResolveConfigPath(args []string) string {
	if len(args) > 1 && args[1] != "" {
		return args[1]
	}

	if path := os.Getenv("AUTH_CONFIG_PATH"); path != "" {
		return path
	}

	return DefaultConfigPath
}
