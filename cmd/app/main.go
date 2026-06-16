// Package main is the entry point for the tick-sync service.
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
		fatal(logger, stop, "load config", err)
	}

	application, err := app.New(ctx, cfg, app.WithLogger(logger))
	if err != nil {
		fatal(logger, stop, "create app", err)
	}
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			logger.Warn("cleanup failed", "error", closeErr)
		}
	}()

	logger.Info("sync service started", "poll_interval", cfg.PollInterval)
	if runErr := application.Run(ctx); runErr != nil {
		fatal(logger, stop, "app run failed", runErr)
	}
}

// fatal logs the error, runs deferred cleanup that os.Exit would otherwise skip, and exits.
func fatal(logger *slog.Logger, stop func(), msg string, err error) {
	logger.Error(msg, "error", err)
	stop()
	os.Exit(1)
}
