package googletasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

const querySelectSyncState = `
SELECT google_task_id, ticktick_task_id, post_sync_action, google_finalized_at
FROM synced_google_tasks
WHERE google_task_id = ?
LIMIT 1;`

// GetSyncState returns the persisted sync state for a Google task.
func (r *Repo) GetSyncState(ctx context.Context, googleTaskID string) (googletasksync.SyncState, bool, error) {
	var state googletasksync.SyncState
	var finalizedAt string
	err := r.db.QueryRowContext(ctx, querySelectSyncState, googleTaskID).Scan(
		&state.GoogleTaskID,
		&state.TickTickTaskID,
		&state.PostSyncAction,
		&finalizedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return googletasksync.SyncState{}, false, nil
	}
	if err != nil {
		return googletasksync.SyncState{}, false, fmt.Errorf("get google task sync state %s: %w", googleTaskID, err)
	}

	if finalizedAt != "" {
		state.GoogleFinalizedAt, err = parseTime(finalizedAt)
		if err != nil {
			return googletasksync.SyncState{}, false, fmt.Errorf("parse google task finalization time: %w", err)
		}
	}

	return state, true, nil
}
