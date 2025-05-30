package main

import (
	"context"
	"fmt"

	"GolangTemplateProject/config"
	"GolangTemplateProject/internal/usecase/authorization"
	"GolangTemplateProject/internal/user/delivery/http"
	"GolangTemplateProject/internal/user/repository/postgresql"
	"GolangTemplateProject/pkg/adapters/postgres"
	open_telemetry "GolangTemplateProject/pkg/open-telemetry"
	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
)

// @title ProjectAPI
// @version 1.0
// @description This is a sample swagger for Fiber project
// @contact.name Andrey
// @contact.email damdinov@jcraster.ru
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:10001
// @BasePath /
func main() {

	ctx := context.Background()
	err := config.LoadConfig()
	fmt.Println(config.Get())
	if err != nil {
		panic(err)
	}

	shutdowns, err := open_telemetry.SetupOpenTelemetrySDK(ctx)
	if err != nil {
		panic(err)
	}
	postgres.New(ctx)

	usecaseUser := authorization.NewUserUsecase(postgresql.NewUserRepository(database))
	service := http.NewUserService(usecaseUser)

	app := fiber.New()

	http.AddToRouter(app, service)
	//err = app.ListenTLS(
	//	config.Get().ServerInfo.HttpConnection.String(),
	//	config.Get().ServerInfo.TLS.Cert,
	//	config.Get().ServerInfo.TLS.Key,
	//)

	if err := app.Listen(config.Get().ServerInfo.HttpConnection.String()); err != nil {
		panic(err)
	}
	_ = shutdowns(ctx)
}
