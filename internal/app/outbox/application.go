package outbox

import (
	"context"
	"time"

	"GolangTemplateProject/config"
	"GolangTemplateProject/internal/usecase/outbox"
	"GolangTemplateProject/pkg/adapters/kafka"
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
	ctx := context.Background()
	usecaseUser := outbox.NewUserUsecase(nil)
	//impl, _ := consumer.NewTopicConsumerGroup[domain.User](nil, nil, usecaseUser.Translate)
	//user.NewUserListener(impl, usecaseUser)
	scheduler, err := jobs.NewJobScheduler(dlog.New(), nil)
	if err != nil {
		panic(err)
	}
	scheduler.AddJob(
		jobs.NewJobBuilder(
			jobs.DurationJob(time.Second)).
			SetTask(usecaseUser.Translate, ctx).
			SetOptions(gocron.WithName("SaveSignedDocumentStream")),
	)
	a.gracefulCloser.AddCloser(scheduler.Stop)
	scheduler.Start()
	select {
	case <-ctx.Done():
		a.gracefulCloser.Close()
	}
}
