package http

import (
	"GolangTemplateProject/config"
	"GolangTemplateProject/internal/user"
	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"go.opentelemetry.io/otel"
)

func SpanNameStrategy(c *fiber.Ctx) string {
	return c.Method() + " " + c.Path()
}

func AddToRouter(fiberApp fiber.Router, handler user.HttpHandler) {

	fiberApp.Get("/swagger/*", swagger.HandlerDefault) // default

	fiberApp.Get("/swagger/*", swagger.New(swagger.Config{ // custom
		URL:         "http://example.com/doc.json",
		DeepLinking: false,
		// Expand ("list") or Collapse ("none") tag groups by default
		DocExpansion: "none",
		// Prefill OAuth ClientId on Authorize popup
		OAuth: &swagger.OAuthConfig{
			AppName:  "OAuth Provider",
			ClientId: "21bb4edc-05a7-4afc-86f1-2e151e4ba6e2",
		},
		// Ability to change OAuth2 redirect uri location
		OAuth2RedirectUrl: "http://localhost:8080/swagger/oauth2-redirect.html",
	}))

	fiberApp.Use(otelfiber.Middleware(
		otelfiber.WithServerName(config.Get().ServerInfo.Name),
		otelfiber.WithTracerProvider(otel.GetTracerProvider()),
		//otelfiber.WithMeterProvider(otel.GetMeterProvider()),
		//otelfiber.WithPropagators(otel.GetTextMapPropagator()),
		otelfiber.WithSpanNameFormatter(SpanNameStrategy),
	))

	fiberApp.Post("/login", handler.Login())
}
