package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	gtasksclient "github.com/prepin/tick-sync/internal/clients/googletasks"
	ticktickclient "github.com/prepin/tick-sync/internal/clients/ticktick"
	"github.com/prepin/tick-sync/internal/config"
	googletasksrepo "github.com/prepin/tick-sync/internal/infra/sqlite/googletasks"
	googletaskssyncjob "github.com/prepin/tick-sync/internal/jobs/googletaskssync"
	usecase "github.com/prepin/tick-sync/internal/usecases/googletasksync"
	_ "modernc.org/sqlite"
)

type Runner interface {
	Start(ctx context.Context)
}

type Option func(*App)

type App struct {
	cfg  config.Config
	db   *sql.DB
	jobs []Runner
}

func WithJobs(jobs []Runner) Option {
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

	repo, err := googletasksrepo.NewGoogleTasksRepo(ctx, db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create google tasks repo: %w", err)
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

	uc := usecase.New(google, ticktick, repo, cfg.GooglePostSyncAction)
	a.jobs = []Runner{googletaskssyncjob.New(uc, cfg.PollInterval)}

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
