package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	googleclient "github.com/prepin/tick-sync/internal/clients/google"
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

type App struct {
	cfg  config.Config
	db   *sql.DB
	jobs []Runner
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	repo, err := googletasksrepo.NewGoogleTasksRepo(ctx, db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create google tasks repo: %w", err)
	}

	google, err := googleclient.New(ctx, cfg)
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
	job := googletaskssyncjob.New(uc, cfg.PollInterval)

	return &App{
		cfg:  cfg,
		db:   db,
		jobs: []Runner{job},
	}, nil
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
