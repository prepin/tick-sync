// Package app orchestrates the tick-sync service lifecycle.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	// Register the modernc.org/sqlite database driver.
	_ "modernc.org/sqlite"

	"github.com/prepin/tick-sync/internal/config"
	googletasksyncjob "github.com/prepin/tick-sync/internal/entrypoints/cron/googletasksync"
	"github.com/prepin/tick-sync/internal/entrypoints/httpserver"
	googletasks "github.com/prepin/tick-sync/internal/infra/clients/googletasks"
	ticktick "github.com/prepin/tick-sync/internal/infra/clients/ticktick"
	googletasksrepo "github.com/prepin/tick-sync/internal/infra/sqlite/googletasks"
	sqlitemigrate "github.com/prepin/tick-sync/internal/infra/sqlite/migrate"
	"github.com/prepin/tick-sync/internal/infra/sqlite/tickticktokens"
	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

// JobsRunner defines the interface for background jobs started by App.
type JobsRunner interface {
	Start(ctx context.Context)
}

// Option configures an App during construction.
type Option func(*App)

// App ties together configuration, storage, clients, and background jobs.
type App struct {
	cfg    config.Config
	db     *sql.DB
	jobs   []JobsRunner
	web    JobsRunner
	logger *slog.Logger
}

// WithJobs overrides the default jobs created by New.
func WithJobs(jobs []JobsRunner) Option {
	return func(a *App) {
		a.jobs = jobs
	}
}

// WithLogger configures the logger used by the app.
func WithLogger(logger *slog.Logger) Option {
	return func(a *App) {
		a.logger = logger
	}
}

// New creates an App with the given configuration and options.
func New(ctx context.Context, cfg config.Config, opts ...Option) (*App, error) {
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if err := sqlitemigrate.Up(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run sqlite migrations: %w", err)
	}

	a := &App{cfg: cfg, db: db, logger: slog.New(slog.DiscardHandler)}
	for _, opt := range opts {
		opt(a)
	}

	if a.jobs != nil {
		tokenRepo, err := tickticktokens.New(db)
		if err != nil {
			if closeErr := db.Close(); closeErr != nil {
				a.logger.WarnContext(ctx, "close db after ticktick token repo init failure", "error", closeErr)
			}
			return nil, fmt.Errorf("create ticktick token repo: %w", err)
		}
		a.web = httpserver.New(cfg, tokenRepo, httpserver.WithLogger(a.logger))
		return a, nil
	}

	repo, err := googletasksrepo.New(ctx, db)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			a.logger.WarnContext(ctx, "close db after repo init failure", "error", closeErr)
		}
		return nil, fmt.Errorf("create google tasks repo: %w", err)
	}

	google, err := googletasks.New(ctx, cfg)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			a.logger.WarnContext(ctx, "close db after google client init failure", "error", closeErr)
		}
		return nil, fmt.Errorf("create google tasks client: %w", err)
	}

	tokenRepo, err := tickticktokens.New(db)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			a.logger.WarnContext(ctx, "close db after ticktick token repo init failure", "error", closeErr)
		}
		return nil, fmt.Errorf("create ticktick token repo: %w", err)
	}

	ticktick, err := ticktick.New(cfg, tokenRepo)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			a.logger.WarnContext(ctx, "close db after ticktick client init failure", "error", closeErr)
		}
		return nil, fmt.Errorf("create ticktick client: %w", err)
	}

	uc := googletasksync.New(
		google,
		ticktick,
		repo,
		cfg.GooglePostSyncAction,
		googletasksync.WithTodayImportDelay(cfg.GoogleTodayImportDelay),
		googletasksync.WithLocation(cfg.Location),
	)
	a.jobs = []JobsRunner{googletasksyncjob.New(uc, cfg.PollInterval, googletasksyncjob.WithLogger(a.logger))}
	a.web = httpserver.New(cfg, tokenRepo, httpserver.WithLogger(a.logger))

	return a, nil
}

// Run starts all background jobs and blocks until the context is cancelled.
func (a *App) Run(ctx context.Context) error {
	a.logger.InfoContext(ctx, "sync service started", "poll_interval", a.cfg.PollInterval)

	for _, job := range a.jobs {
		job.Start(ctx)
	}
	if a.web != nil {
		a.web.Start(ctx)
	}

	<-ctx.Done()
	a.logger.InfoContext(ctx, "shutdown requested")
	return nil
}

// Close releases the database connection.
func (a *App) Close() error {
	return a.db.Close()
}
