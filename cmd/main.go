package main

import (
	"context"
	"fmt"

	"GolangTemplateProject/config"
	"GolangTemplateProject/internal/user/delivery/http"
	"GolangTemplateProject/internal/user/usecase"
	open_telemetry "GolangTemplateProject/pkg/open-telemetry"
	"github.com/gofiber/fiber/v2"
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

	app := fiber.New()

	service := http.NewUserService(&usecase.UserUsecase{})
	http.AddToRouter(app, service)
	err = app.ListenTLS(
		config.Get().ServerInfo.HttpConnection.String(),
		config.Get().ServerInfo.TLS.Cert,
		config.Get().ServerInfo.TLS.Key,
	)

	if err := app.Listen(config.Get().ServerInfo.HttpConnection.String()); err != nil {
		panic(err)
	}
	_ = shutdowns(ctx)
}
