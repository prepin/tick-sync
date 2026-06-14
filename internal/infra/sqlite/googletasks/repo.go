package googletasks

import (
	"context"
	"database/sql"
	"fmt"
	"time"
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

const querySelectSyncedTaskExists = `
SELECT 1
FROM synced_google_tasks
WHERE google_task_id = ?
LIMIT 1;`

const queryInsertSyncedTask = `
INSERT OR IGNORE INTO synced_google_tasks (
  google_task_id,
  google_updated,
  google_title,
  ticktick_task_id,
  post_sync_action,
  synced_at
) VALUES (?, ?, ?, ?, ?, ?);`

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

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
