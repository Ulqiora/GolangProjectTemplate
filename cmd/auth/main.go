package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"GolangTemplateProject/internal/app/authsvc"
)

func main() {
	configPath := authsvc.ResolveConfigPath(os.Args)
	app, err := authsvc.New(configPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err = app.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
