package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"GolangTemplateProject/internal/app/authsvc"
	"GolangTemplateProject/internal/app/outboxsvc"
)

func main() {
	configPath := authsvc.ResolveConfigPath(os.Args)
	application, err := outboxsvc.New(configPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err = application.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
