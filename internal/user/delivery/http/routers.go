package http

import (
	"GolangTemplateProject/config"
	"GolangTemplateProject/internal/user"
	"github.com/gofiber/contrib/otelfiber"
	swagg "github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
)

func SpanNameStrategy(c *fiber.Ctx) string {
	return c.Method() + " " + c.Path()
}

func AddToRouter(fiberApp fiber.Router, handler user.HttpHandler) {
	cfg := swagg.Config{
		BasePath: "/",
		FilePath: "./docs/swagger.json",
		Path:     "swagger",
		Title:    "Swagger API Docs",
	}

	fiberApp.Use(swagg.New(cfg))

	fiberApp.Use(otelfiber.Middleware(
		otelfiber.WithServerName(config.Get().ServerInfo.Name),
		otelfiber.WithTracerProvider(otel.GetTracerProvider()),
		//otelfiber.WithMeterProvider(otel.GetMeterProvider()),
		//otelfiber.WithPropagators(otel.GetTextMapPropagator()),
		otelfiber.WithSpanNameFormatter(SpanNameStrategy),
	))

	fiberApp.Post("/login", handler.Login())
}
