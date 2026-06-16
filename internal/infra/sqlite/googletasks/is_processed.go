package googletasks

import (
	"context"
	"database/sql"
	"fmt"
)

const querySelectSyncedTaskExists = `
SELECT 1
FROM synced_google_tasks
WHERE google_task_id = ?
LIMIT 1;`

// IsProcessed returns true if the Google task ID has already been synced.
func (r *Repo) IsProcessed(ctx context.Context, googleTaskID string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, querySelectSyncedTaskExists, googleTaskID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}

	return false, fmt.Errorf("check synced google task %s: %w", googleTaskID, err)
}
