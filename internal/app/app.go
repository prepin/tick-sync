package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	googletasksync "github.com/prepin/tick-sync/internal/application/googletasksync"
	"github.com/prepin/tick-sync/internal/config"
	gtasksclient "github.com/prepin/tick-sync/internal/infra/googletasks"
	gtasksrepo "github.com/prepin/tick-sync/internal/infra/sqlite/syncedtasks"
	ticktickclient "github.com/prepin/tick-sync/internal/infra/ticktick"
	googletasksyncjob "github.com/prepin/tick-sync/internal/transport/cron/googletasksync"
	_ "modernc.org/sqlite"
)

type JobsRunner interface {
	Start(ctx context.Context)
}

type Option func(*App)

type App struct {
	cfg  config.Config
	db   *sql.DB
	jobs []JobsRunner
}

func WithJobs(jobs []JobsRunner) Option {
	return func(a *App) {
		a.jobs = jobs
	}
}

func New(ctx context.Context, cfg config.Config, opts ...Option) (*App, error) {
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	a := &App{cfg: cfg, db: db}
	for _, opt := range opts {
		opt(a)
	}

	if a.jobs != nil {
		return a, nil
	}

	repo, err := gtasksrepo.New(ctx, db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create synced tasks repo: %w", err)
	}

	google, err := gtasksclient.New(ctx, cfg)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create google tasks client: %w", err)
	}

	ticktick, err := ticktickclient.New(cfg)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create ticktick client: %w", err)
	}

	uc := googletasksync.New(google, ticktick, repo, cfg.GooglePostSyncAction)
	a.jobs = []JobsRunner{googletasksyncjob.New(uc, cfg.PollInterval)}

	return a, nil
}

func (a *App) Run(ctx context.Context) error {
	slog.Info("sync service started", "poll_interval", a.cfg.PollInterval)

	for _, job := range a.jobs {
		job.Start(ctx)
	}

	<-ctx.Done()
	slog.Info("shutdown requested")
	return nil
}

func (a *App) Close() error {
	return a.db.Close()
}
