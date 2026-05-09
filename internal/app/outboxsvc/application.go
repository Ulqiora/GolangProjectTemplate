package outboxsvc

import (
	"context"
	"fmt"

	kafkaadapter "GolangTemplateProject/internal/adapters/secondary/kafka"
	outboxworker "GolangTemplateProject/internal/application/outbox"
	"GolangTemplateProject/internal/config"
	"GolangTemplateProject/internal/repository/outboxmessages"
	"GolangTemplateProject/pkg/adapters/postgres"
	cryptobcrypt "GolangTemplateProject/pkg/cripto/bcrypt"
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
	publisher     *kafkaadapter.OutboxPublisher
	worker        *outboxworker.Worker
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
	if a.worker == nil {
		return fmt.Errorf("outbox worker is not configured")
	}

	go a.worker.Run(ctx)
	<-ctx.Done()
	return a.Close(context.Background())
}

func (a *Application) Close(ctx context.Context) error {
	var closeErr error
	if a.publisher != nil {
		if err := a.publisher.Close(); err != nil && closeErr == nil {
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

	appLogger, err := logger.NewLogger(a.cfg.Env, otel.GetTracerProvider().Tracer(a.cfg.ServiceName+":outbox"))
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	logger.SetDefaultLogger(appLogger)
	a.log = appLogger.WithN("outbox_app")

	// Touch bcrypt config here so broken runtime config is caught before deploy.
	_ = cryptobcrypt.New(a.cfg.Auth.Bcrypt)

	db, err := postgres.NewWithOptions(ctx, &a.cfg.Database.Postgres, postgres.WithLogger(a.log), postgres.WithTracing())
	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}
	a.postgres = db

	publisher, err := kafkaadapter.NewOutboxPublisher(a.cfg.AuthService.KafkaProducer, a.log)
	if err != nil {
		return fmt.Errorf("init kafka publisher: %w", err)
	}
	a.publisher = publisher

	txManager := transactionmanager.NewWithOptions(db, transactionmanager.WithLogger(a.log))
	outboxRepo := outboxmessages.New(db)
	a.worker = outboxworker.NewWorker(txManager, outboxRepo, publisher, a.log, nil)

	a.log.Info("Outbox dispatcher configured")
	return nil
}
