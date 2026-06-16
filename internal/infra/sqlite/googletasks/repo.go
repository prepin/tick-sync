package googletasks

import (
	"context"
	"database/sql"
	"errors"
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

// Repo is the SQLite implementation of SyncedTaskRepository.
type Repo struct {
	db *sql.DB
}

// New creates a Repo and ensures the required table exists.
func New(ctx context.Context, db *sql.DB) (*Repo, error) {
	if db == nil {
		return nil, errors.New("google tasks repo: db is nil")
	}

	if _, err := db.ExecContext(ctx, createSyncedGoogleTasksTable); err != nil {
		return nil, fmt.Errorf("create synced_google_tasks table: %w", err)
	}

	return &Repo{db: db}, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
