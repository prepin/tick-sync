package service

import (
	"context"
	"database/sql"
	"fmt"
	"io"
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
	out     io.Writer
}

func NewSyncRunner(ctx context.Context, cfg config.Config, out io.Writer) (*SyncRunner, func() error, error) {
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
		out:     out,
	}

	return runner, cleanup, nil
}

func (r *SyncRunner) RunOnce(ctx context.Context) error {
	summary, syncErr := r.usecase.SyncGoogleTasksToTickTick(ctx)
	PrintSyncSummary(r.out, summary)
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

func PrintSyncSummary(out io.Writer, summary googletasksync.SyncSummary) {
	fmt.Fprintln(out, "Sync summary:")
	fmt.Fprintf(out, "Seen: %d\n", summary.Seen)
	fmt.Fprintf(out, "Created: %d\n", summary.Created)
	fmt.Fprintf(out, "Skipped: %d\n", summary.Skipped)
	fmt.Fprintf(out, "Failed: %d\n", summary.Failed)
	fmt.Fprintf(out, "Completed: %d\n", summary.Completed)
	fmt.Fprintf(out, "Deleted: %d\n", summary.Deleted)

	if len(summary.Errors) == 0 {
		return
	}

	fmt.Fprintln(out, "Errors:")
	for _, err := range summary.Errors {
		if err == nil {
			continue
		}
		fmt.Fprintf(out, "- %v\n", err)
	}
}
