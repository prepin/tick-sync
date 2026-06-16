package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/prepin/tick-sync/internal/app"
	"github.com/prepin/tick-sync/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	application, err := app.New(ctx, cfg, app.WithLogger(logger))
	if err != nil {
		logger.Error("create app", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			logger.Warn("cleanup failed", "error", closeErr)
		}
	}()

	logger.Info("sync service started", "poll_interval", cfg.PollInterval)
	if runErr := application.Run(ctx); runErr != nil {
		logger.Error("app run failed", "error", runErr)
		os.Exit(1)
	}
}
