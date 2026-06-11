package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/service"
)

type syncRunner interface {
	RunOnce(ctx context.Context) error
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	runner, cleanup, err := service.NewSyncRunner(ctx, cfg)
	if err != nil {
		slog.Error("create sync runner", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := cleanup(); err != nil {
			slog.Warn("cleanup failed", "error", err)
		}
	}()

	slog.Info("sync service started", "poll_interval", cfg.PollInterval)
	if err := runSync(ctx, runner); err != nil {
		os.Exit(1)
	}

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("shutdown requested")
			return
		case <-ticker.C:
			runSync(ctx, runner)
		}
	}
}

func runSync(ctx context.Context, runner syncRunner) error {
	slog.Info("sync started")
	if err := runner.RunOnce(ctx); err != nil {
		slog.Error("sync finished", "error", err)
		return err
	}
	slog.Info("sync finished")
	return nil
}
