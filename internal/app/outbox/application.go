package outbox

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"GolangTemplateProject/config"
	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/internal/repository/user"
	"GolangTemplateProject/internal/usecase/outbox"
	"GolangTemplateProject/pkg/adapters/kafka"
	sync_producer "GolangTemplateProject/pkg/adapters/kafka/producer/sync-producer"
	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/closer"
	"GolangTemplateProject/pkg/jobs"
	open_telemetry "GolangTemplateProject/pkg/open-telemetry"
	"github.com/go-co-op/gocron/v2"
	"gitlab.wildberries.ru/wbbank/go-dpkg/dlog/v1"
)

type Application struct {
	postgres       postgres.IPostgres
	consumer       kafka.Consumer
	gracefulCloser *closer.GracefulCloser
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
	telemetrySDK, err := open_telemetry.SetupOpenTelemetrySDK(ctx)
	if err != nil {
		panic(err)
	}
	a.gracefulCloser.AddCloser(telemetrySDK)
	return nil
}

func (a *Application) Start() {
	ctx, cancel := context.WithCancel(context.Background())

	producer, err := sync_producer.NewTopicProducer(config.Get().Kafka, nil)
	fmt.Println(config.Get().Kafka)
	if err != nil {
		panic(err)
	}

	baseUserRepository := ports.NewBaseRepository(a.postgres, "user", func() *domain.User {
		return &domain.User{}
	})
	userRepository := user.NewUserRepository(baseUserRepository)
	usecaseUser := outbox.NewUserUsecase(userRepository, producer)
	//impl, _ := consumer.NewTopicConsumerGroup[domain.User](nil, nil, usecaseUser.Translate)
	//user.NewUserListener(impl, usecaseUser)
	scheduler, err := jobs.NewJobScheduler(dlog.New(), nil)
	if err != nil {
		panic(err)
	}

	jobCtx, jobCancel := context.WithCancel(ctx)

	scheduler.AddJob(
		jobs.NewJobBuilder(
			jobs.DurationJob(5*time.Second)).
			SetTask(usecaseUser.Translate, jobCtx).
			SetOptions(gocron.WithName("UserTranslate")),
	)
	a.gracefulCloser.AddCloser(func() error {
		jobCancel()
		return nil
	})
	a.gracefulCloser.AddCloser(scheduler.Stop)
	scheduler.Start()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	log.Println("application started, press Ctrl+C to stop")
	<-interrupt
	log.Println("shutting down...")
	if err := a.gracefulCloser.Close(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("forced shutdown: some components didn't stop in time")
		} else {
			log.Println("shutdown error:", err)
		}
	}
	cancel()

	log.Println("application stopped")
}
