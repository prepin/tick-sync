package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	googleclient "github.com/prepin/tick-sync/internal/clients/google"
	ticktickclient "github.com/prepin/tick-sync/internal/clients/ticktick"
	"github.com/prepin/tick-sync/internal/config"
	sqlitestore "github.com/prepin/tick-sync/internal/infra/sqlite"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
	_ "modernc.org/sqlite"
)

type SyncRunner struct {
	usecase *googletasksync.Usecase
}

func NewSyncRunner(ctx context.Context, cfg config.Config) (*SyncRunner, func() error, error) {
	postSyncAction, err := PostSyncActionFromConfig(cfg.GooglePostSyncAction)
	if err != nil {
		return nil, nil, err
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite db: %w", err)
	}
	cleanup := func() error {
		return db.Close()
	}

	store, err := sqlitestore.NewGoogleTasksStore(ctx, db)
	if err != nil {
		_ = cleanup()
		return nil, nil, fmt.Errorf("create google tasks store: %w", err)
	}

	google, err := googleclient.New(ctx, cfg)
	if err != nil {
		_ = cleanup()
		return nil, nil, fmt.Errorf("create google tasks client: %w", err)
	}

	ticktick, err := ticktickclient.New(cfg)
	if err != nil {
		_ = cleanup()
		return nil, nil, fmt.Errorf("create ticktick client: %w", err)
	}

	runner := &SyncRunner{
		usecase: googletasksync.New(google, ticktick, store, postSyncAction),
	}

	return runner, cleanup, nil
}

func (r *SyncRunner) RunOnce(ctx context.Context) error {
	summary, syncErr := r.usecase.SyncGoogleTasksToTickTick(ctx)
	PrintSyncSummary(summary)
	if syncErr != nil {
		return fmt.Errorf("sync google tasks to ticktick: %w", syncErr)
	}

	return nil
}

func PostSyncActionFromConfig(value string) (googletasksync.PostSyncAction, error) {
	switch strings.TrimSpace(value) {
	case "", "complete":
		return googletasksync.PostSyncActionComplete, nil
	case "delete":
		return googletasksync.PostSyncActionDelete, nil
	default:
		return "", fmt.Errorf("unsupported GOOGLE_POST_SYNC_ACTION %q; expected complete or delete", value)
	}
}

func PrintSyncSummary(summary googletasksync.SyncSummary) {
	attrs := []slog.Attr{
		slog.Int("seen", summary.Seen),
		slog.Int("created", summary.Created),
		slog.Int("skipped", summary.Skipped),
		slog.Int("failed", summary.Failed),
		slog.Int("completed", summary.Completed),
		slog.Int("deleted", summary.Deleted),
	}

	if len(summary.Errors) > 0 {
		nonNil := make([]string, 0, len(summary.Errors))
		for _, err := range summary.Errors {
			if err != nil {
				nonNil = append(nonNil, err.Error())
			}
		}
		if len(nonNil) > 0 {
			attrs = append(attrs, slog.String("errors", strings.Join(nonNil, ", ")))
		}
	}

	if summary.Failed > 0 || len(summary.Errors) > 0 {
		slog.LogAttrs(context.Background(), slog.LevelError, "sync finished", attrs...)
	} else {
		slog.LogAttrs(context.Background(), slog.LevelInfo, "sync finished", attrs...)
	}
}
