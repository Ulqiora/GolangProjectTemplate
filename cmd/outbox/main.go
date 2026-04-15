package main

import (
	"context"
	"log/slog"

	"GolangTemplateProject/internal/app/outbox"
)

func main() {
	var ctx = context.Background()
	application, err := outbox.NewApplication(ctx)
	if err != nil {
		slog.Error(err.Error())
	}
	if err = application.SetupDependencies(ctx); err != nil {
		slog.Error(err.Error())
	}
	application.Start()
}
