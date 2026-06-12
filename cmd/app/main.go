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

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	application, err := app.New(ctx, cfg)
	if err != nil {
		slog.Error("create app", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := application.Close(); err != nil {
			slog.Warn("cleanup failed", "error", err)
		}
	}()

	slog.Info("sync service started", "poll_interval", cfg.PollInterval)
	if err := application.Run(ctx); err != nil {
		slog.Error("app run failed", "error", err)
		os.Exit(1)
	}
}
