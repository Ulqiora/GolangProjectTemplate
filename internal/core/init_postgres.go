package core

import (
	"time"

	"GolangTemplateProject/config"
	"GolangTemplateProject/pkg/adapters/postgres"
	"github.com/jmoiron/sqlx"
)

type ServiceDependencies struct {
	pool postgres.IPostgres
}

func ConnectToDatabase(dependecies *ServiceDependencies) error {
	connectionURL := config.Get().Database.Postgres.ConnectionString()
	database, err := sqlx.Connect("postgres", connectionURL)
	if err != nil {
		return err
	}
	database.SetConnMaxLifetime(time.Duration(config.Get().Database.Postgres.Settings.ConnMaxLifetime) * time.Second)
	database.SetConnMaxIdleTime(time.Duration(config.Get().Database.Postgres.Settings.ConnMaxIdleTime) * time.Second)
	database.SetMaxIdleConns(config.Get().Database.Postgres.Settings.MaxIdleConnections)
	database.SetMaxOpenConns(config.Get().Database.Postgres.Settings.MaxOpenConnections)
	dependecies.pool = database
	return nil
}
