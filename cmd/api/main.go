package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"marketplace-backend/internal/app"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := observability.NewLogger(cfg.LogLevel).With("service", "marketplace-api", "env", cfg.Env)
	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("build app", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := application.Run(ctx); err != nil {
		logger.Error("run app", "error", err)
		os.Exit(1)
	}
}
