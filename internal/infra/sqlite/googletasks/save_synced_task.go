package googletasks

import (
	"context"
	"fmt"

	"github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

const queryInsertSyncedTask = `
INSERT OR IGNORE INTO synced_google_tasks (
  google_task_id,
  google_updated,
  google_title,
  ticktick_task_id,
  post_sync_action,
  synced_at,
  google_finalized_at
) VALUES (?, ?, ?, ?, ?, ?, ?);`

// SaveSyncedTask records a synced task mapping in SQLite.
func (r *Repo) SaveSyncedTask(ctx context.Context, params googletasksync.SaveSyncedTaskParams) error {
	_, err := r.db.ExecContext(ctx, queryInsertSyncedTask,
		params.GoogleTaskID,
		params.GoogleUpdated,
		params.GoogleTitle,
		params.TickTickTaskID,
		string(params.PostSyncAction),
		formatTime(params.SyncedAt),
		"",
	)
	if err != nil {
		return fmt.Errorf("save synced google task %s: %w", params.GoogleTaskID, err)
	}

	return nil
}
