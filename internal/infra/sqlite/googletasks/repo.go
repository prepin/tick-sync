package googletasks

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
)

const createSyncedGoogleTasksTable = `
CREATE TABLE IF NOT EXISTS synced_google_tasks (
  google_task_id TEXT PRIMARY KEY,
  google_updated TEXT,
  google_title TEXT,
  ticktick_task_id TEXT NOT NULL,
  post_sync_action TEXT NOT NULL,
  synced_at TEXT NOT NULL
);`

var _ googletasksync.SyncRepo = (*GoogleTasksRepo)(nil)

type GoogleTasksRepo struct {
	db *sql.DB
}

func NewGoogleTasksRepo(ctx context.Context, db *sql.DB) (*GoogleTasksRepo, error) {
	if db == nil {
		return nil, fmt.Errorf("google tasks repo: db is nil")
	}

	if _, err := db.ExecContext(ctx, createSyncedGoogleTasksTable); err != nil {
		return nil, fmt.Errorf("create synced_google_tasks table: %w", err)
	}

	return &GoogleTasksRepo{db: db}, nil
}

func (r *GoogleTasksRepo) IsProcessed(ctx context.Context, googleTaskID string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `
SELECT 1
FROM synced_google_tasks
WHERE google_task_id = ?
LIMIT 1;`, googleTaskID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}

	return false, fmt.Errorf("check synced google task %s: %w", googleTaskID, err)
}

func (r *GoogleTasksRepo) SaveSyncedTask(ctx context.Context, record googletasksync.SyncedTaskRecord) error {
	_, err := r.db.ExecContext(ctx, `
INSERT OR IGNORE INTO synced_google_tasks (
  google_task_id,
  google_updated,
  google_title,
  ticktick_task_id,
  post_sync_action,
  synced_at
) VALUES (?, ?, ?, ?, ?, ?);`,
		record.GoogleTaskID,
		record.GoogleUpdated,
		record.GoogleTitle,
		record.TickTickTaskID,
		string(record.PostSyncAction),
		formatTime(record.SyncedAt),
	)
	if err != nil {
		return fmt.Errorf("save synced google task %s: %w", record.GoogleTaskID, err)
	}

	return nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
