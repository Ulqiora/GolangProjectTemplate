package app

import (
	"context"

	"GolangTemplateProject/config"
	"GolangTemplateProject/pkg/adapters/postgres"
)

type Application struct {
	postgres postgres.IPostgres
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
	err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	pool, err := postgres.New(ctx, &config.Get().Database.Postgres)
	a.postgres = pool
	if err != nil {
		panic(err)
	}
	return nil
}
