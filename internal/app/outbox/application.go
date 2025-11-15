package outbox

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GolangTemplateProject/internal/adapters/primary/k8s"
	"GolangTemplateProject/internal/adapters/primary/prometheus"
	auth_handlers "GolangTemplateProject/internal/adapters/primary/user"
	"GolangTemplateProject/internal/config"
	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	user_repo "GolangTemplateProject/internal/repository/user"
	user_secret_repo "GolangTemplateProject/internal/repository/user-secrets"
	auth_usecase "GolangTemplateProject/internal/usecase/authorization"
	"GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/closer"
	"GolangTemplateProject/pkg/jobs"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/smart-span/tracing"
	transaction_manager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/gin-gonic/gin"
	"gitlab.wildberries.ru/wbbank/go-dpkg/dlog/v1"
	"go.opentelemetry.io/otel/trace"
)

type Application struct {
	postgres       postgres.IPostgres
	consumer       kafka.Consumer
	gracefulCloser *closer.GracefulCloser
	scheduler      jobs.Scheduler
	tracerProvider trace.TracerProvider
	logger         logger.Logger
}

func NewApplication(ctx context.Context) (*Application, error) {
	app := new(Application)
	err := app.SetupDependencies(ctx)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (a *Application) SetupDependencies(ctx context.Context) error {
	a.gracefulCloser = closer.NewGracefulCloser()
	err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	pool, err := postgres.New(ctx, &config.Get().Database.Postgres)
	a.postgres = pool
	if err != nil {
		panic(err)
	}
	a.gracefulCloser.AddCloser(pool.Close)

	a.scheduler, err = jobs.NewJobScheduler(dlog.New(), nil)
	if err != nil {
		panic(err)
	}
	a.gracefulCloser.AddCloser(a.scheduler.Stop)

	tracerProvider, f, err := tracing.InitTracing(ctx, &config.Get().Trace.Jaeger)
	a.gracefulCloser.AddCloser(func() error {
		return f(ctx)
	})
	a.logger, err = logger.NewLogger()
	logger.SetDefaultLogger(a.logger)
	if err != nil {
		panic(err)
	}
	a.tracerProvider = tracerProvider
	return nil
}

func (a *Application) Start() {
	//ctx, cancel := context.WithCancel(context.Background())

	//producer, err := sync_producer.NewTopicProducer(config.Get().Kafka, nil)
	//fmt.Println(config.Get().Kafka)
	//if err != nil {
	//	panic(err)
	//}

	trxManager := transaction_manager.New(a.postgres, a.logger)

	baseUserRepository := ports.NewBaseRepository[*domain.User](a.postgres, "user_registration", func() *domain.User {
		return &domain.User{}
	})
	baseUserSecretRepository := ports.NewBaseRepository[*domain.UserSecrets](a.postgres, "user_secret", func() *domain.UserSecrets {
		return &domain.UserSecrets{}
	})
	userRepository := user_repo.NewUserRepository(baseUserRepository)
	userSecretRepository := user_secret_repo.NewUserSecretRepository(baseUserSecretRepository)
	usecaseUser := auth_usecase.NewUserUsecase(userRepository, userSecretRepository, trxManager)

	//jobCtx, jobCancel := context.WithCancel(ctx)
	//a.scheduler.AddJob(
	//	jobs.NewJobBuilder(
	//		jobs.DurationJob(5*time.Second)).
	//		SetTask(usecaseUser.Translate, jobCtx).
	//		SetOptions(gocron.WithName("UserTranslate")),
	//)
	//a.gracefulCloser.AddCloser(func() error {
	//	jobCancel()
	//	return nil
	//})

	engine := gin.Default()

	k8s.NewModulePrometheus().Register(engine)
	prometheus.NewModulePromehteus().Register(engine.Group("/metrics"))
	auth_handlers.NewUserHandlers(usecaseUser).Register(engine.Group("/users"))

	srv := &http.Server{
		Addr:    config.Get().ServerInfo.HttpConnection.String(),
		Handler: engine,
	}

	go func() {
		fmt.Printf("Service started on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	a.gracefulCloser.AddCloser(func() error {
		ctxWithTimeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelFunc()
		return srv.Shutdown(ctxWithTimeout)
	})

	a.scheduler.Start()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	<-interrupt
	log.Println("shutting down...")
	if err := a.gracefulCloser.Close(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("forced shutdown: some components didn't stop in time")
		} else {
			log.Println("shutdown error:", err)
		}
	}
	//cancel()
}
