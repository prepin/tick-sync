package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	runner, cleanup, err := service.NewSyncRunner(ctx, cfg, os.Stdout)
	if err != nil {
		log.Fatalf("create sync runner: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			log.Printf("cleanup failed: %v", err)
		}
	}()

	log.Printf("sync service started; poll interval: %s", cfg.PollInterval)
	runSync(ctx, runner)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("shutdown requested")
			return
		case <-ticker.C:
			runSync(ctx, runner)
		}
	}
}

func runSync(ctx context.Context, runner *service.SyncRunner) {
	log.Println("sync started")
	if err := runner.RunOnce(ctx); err != nil {
		log.Printf("sync finished with error: %v", err)
		return
	}
	log.Println("sync finished")
}
