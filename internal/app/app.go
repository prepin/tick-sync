// Package app orchestrates the tick-sync service lifecycle.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	// Register the modernc.org/sqlite database driver.
	_ "modernc.org/sqlite"

	"github.com/prepin/tick-sync/internal/config"
	googletasksyncjob "github.com/prepin/tick-sync/internal/entrypoints/cron/googletasksync"
	tickticktokenreminderjob "github.com/prepin/tick-sync/internal/entrypoints/cron/tickticktokenreminder"
	"github.com/prepin/tick-sync/internal/entrypoints/httpserver"
	googletasks "github.com/prepin/tick-sync/internal/infra/clients/googletasks"
	ticktick "github.com/prepin/tick-sync/internal/infra/clients/ticktick"
	googletasksrepo "github.com/prepin/tick-sync/internal/infra/sqlite/googletasks"
	sqlitemigrate "github.com/prepin/tick-sync/internal/infra/sqlite/migrate"
	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
	"github.com/prepin/tick-sync/internal/usecase/tickticktokenreminder"
)

// Runner defines a long-running app component supervised by App.
type Runner interface {
	Run(ctx context.Context) error
}

// Option configures an App during construction.
type Option func(*App)

// App ties together configuration, storage, clients, and background jobs.
type App struct {
	cfg          config.Config
	db           *sql.DB
	jobs         []Runner
	jobsOverride bool
	web          Runner
	logger       *slog.Logger
}

// WithJobs overrides the default jobs created by New.
func WithJobs(jobs []Runner) Option {
	return func(a *App) {
		a.jobs = jobs
		a.jobsOverride = true
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
	a, err := newBaseApp(ctx, cfg, opts...)
	if err != nil {
		return nil, err
	}

	tokenRepo, err := a.configureWeb(ctx)
	if err != nil {
		return nil, err
	}
	if a.jobsOverride {
		return a, nil
	}

	if err := a.configureSync(ctx, tokenRepo); err != nil {
		return nil, err
	}

	return a, nil
}

func newBaseApp(ctx context.Context, cfg config.Config, opts ...Option) (*App, error) {
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if err := sqlitemigrate.Up(ctx, db); err != nil {
		return nil, closeAfterMigrationFailure(db, err)
	}

	a := &App{cfg: cfg, db: db, logger: slog.New(slog.DiscardHandler)}
	for _, opt := range opts {
		opt(a)
	}
	return a, nil
}

func (a *App) configureWeb(ctx context.Context) (*oauthtokens.Repo, error) {
	tokenRepo, err := oauthtokens.New(a.db)
	if err != nil {
		return nil, a.closeDBAfterInitFailure(
			ctx,
			"close db after oauth token repo init failure",
			err,
			"create oauth token repo",
		)
	}
	a.web = httpserver.New(a.cfg, tokenRepo, httpserver.WithLogger(a.logger))
	return tokenRepo, nil
}

func (a *App) configureSync(ctx context.Context, tokenRepo *oauthtokens.Repo) error {
	repo, err := googletasksrepo.New(a.db)
	if err != nil {
		return a.closeDBAfterInitFailure(ctx, "close db after repo init failure", err, "create google tasks repo")
	}

	google, err := googletasks.New(ctx, a.cfg, tokenRepo)
	if err != nil {
		return a.closeDBAfterInitFailure(
			ctx,
			"close db after google client init failure",
			err,
			"create google tasks client",
		)
	}

	ticktick, err := ticktick.New(a.cfg, tokenRepo)
	if err != nil {
		return a.closeDBAfterInitFailure(
			ctx,
			"close db after ticktick client init failure",
			err,
			"create ticktick client",
		)
	}

	uc := googletasksync.New(
		google,
		ticktick,
		repo,
		a.cfg.GooglePostSyncAction,
		googletasksync.WithTodayImportDelay(a.cfg.GoogleTodayImportDelay),
		googletasksync.WithLocation(a.cfg.Location),
	)
	reminderUC := tickticktokenreminder.New(tokenRepo, ticktick)
	a.jobs = []Runner{
		googletasksyncjob.New(uc, a.cfg.PollInterval, googletasksyncjob.WithLogger(a.logger)),
		tickticktokenreminderjob.New(
			reminderUC,
			a.cfg.TickTickReminderInterval,
			tickticktokenreminderjob.WithLogger(a.logger),
		),
	}
	return nil
}

func closeAfterMigrationFailure(db *sql.DB, err error) error {
	if closeErr := db.Close(); closeErr != nil {
		return fmt.Errorf("run sqlite migrations: %w", errors.Join(err, closeErr))
	}
	return fmt.Errorf("run sqlite migrations: %w", err)
}

func (a *App) closeDBAfterInitFailure(ctx context.Context, logMessage string, err error, wrapMessage string) error {
	if closeErr := a.db.Close(); closeErr != nil {
		a.logger.WarnContext(ctx, logMessage, "error", closeErr)
	}
	return fmt.Errorf("%s: %w", wrapMessage, err)
}

// Run starts all background jobs and blocks until the context is cancelled.
func (a *App) Run(ctx context.Context) error {
	a.logger.InfoContext(ctx, "sync service started", "poll_interval", a.cfg.PollInterval)

	runners := make([]Runner, 0, len(a.jobs)+1)
	runners = append(runners, a.jobs...)
	if a.web != nil {
		runners = append(runners, a.web)
	}
	if len(runners) == 0 {
		<-ctx.Done()
		a.logger.InfoContext(ctx, "shutdown requested")
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	resultCh := make(chan runnerResult, len(runners))
	var wg sync.WaitGroup
	for index, runner := range runners {
		wg.Go(func() {
			resultCh <- runnerResult{index: index, err: runner.Run(runCtx)}
		})
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for result := range resultCh {
		if result.err != nil {
			cancel()
			drainRunnerResults(resultCh)
			return fmt.Errorf("runner %d failed: %w", result.index, result.err)
		}
		if runCtx.Err() == nil {
			cancel()
			drainRunnerResults(resultCh)
			return fmt.Errorf("runner %d stopped unexpectedly", result.index)
		}
	}

	a.logger.InfoContext(ctx, "shutdown requested")
	return nil
}

type runnerResult struct {
	index int
	err   error
}

func drainRunnerResults(resultCh <-chan runnerResult) {
	for result := range resultCh {
		_ = result
	}
}

// Close releases the database connection.
func (a *App) Close() error {
	return a.db.Close()
}
