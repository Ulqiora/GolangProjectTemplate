package vkusersvc

import (
	"context"
	"fmt"

	kafkaadapter "GolangTemplateProject/internal/adapters/secondary/kafka"
	vkusers "GolangTemplateProject/internal/application/vkusers"
	"GolangTemplateProject/internal/config"
	"GolangTemplateProject/internal/repository/authusers"
	"GolangTemplateProject/internal/repository/externalidentities"
	"GolangTemplateProject/internal/repository/outboxmessages"
	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
	smarttracing "GolangTemplateProject/pkg/smart-span/tracing"
	transactionmanager "GolangTemplateProject/pkg/transaction-manager"
	"go.opentelemetry.io/otel"
)

type Application struct {
	cfg           *config.ServiceConfig
	log           logger.Logger
	traceShutdown func(context.Context) error
	postgres      *postgres.Postgres
	consumer      *kafkaadapter.VKUserConsumer
}

func New(configPath string) (*Application, error) {
	if configPath == "" {
		configPath = "configs/auth/local.yaml"
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
	if a.consumer == nil {
		return fmt.Errorf("vk user consumer is not configured")
	}
	err := a.consumer.Run(ctx)
	if err != nil && err != context.Canceled {
		_ = a.Close(context.Background())
		return err
	}
	return a.Close(context.Background())
}

func (a *Application) Close(ctx context.Context) error {
	var closeErr error
	if a.consumer != nil {
		if err := a.consumer.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	if a.postgres != nil {
		if err := a.postgres.Close(); err != nil && closeErr == nil {
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
	traceShutdown := func(context.Context) error { return nil }
	if a.cfg.Tracing.Endpoint != "" {
		_, shutdown, err := smarttracing.InitTracing(ctx, &a.cfg.Tracing)
		if err != nil {
			return fmt.Errorf("init tracing: %w", err)
		}
		traceShutdown = shutdown
	}
	a.traceShutdown = traceShutdown

	appLogger, err := logger.NewLogger(a.cfg.Env, otel.GetTracerProvider().Tracer(a.cfg.ServiceName+":vk-consumer"))
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	logger.SetDefaultLogger(appLogger)
	a.log = appLogger.WithN("vk_user_app")

	db, err := postgres.NewWithOptions(ctx, &a.cfg.Database.Postgres, postgres.WithLogger(a.log), postgres.WithTracing())
	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}
	a.postgres = db

	txManager := transactionmanager.NewWithOptions(db, transactionmanager.WithLogger(a.log))
	service := vkusers.NewService(vkusers.Dependencies{
		TxManager:            txManager,
		UserRepo:             authusers.New(db),
		ExternalIdentityRepo: externalidentities.New(db),
		OutboxRepo:           outboxmessages.New(db),
		Log:                  a.log,
	})

	consumer, err := kafkaadapter.NewVKUserConsumer(a.cfg.AuthService.VKUserConsumer, a.log, service)
	if err != nil {
		return fmt.Errorf("init vk user consumer: %w", err)
	}
	a.consumer = consumer
	a.log.Info("VK user consumer configured")
	return nil
}
