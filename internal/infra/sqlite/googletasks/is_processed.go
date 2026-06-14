package googletasks

import (
	"context"
	"database/sql"
	"fmt"
)

func (r *GoogleTasksRepo) IsProcessed(ctx context.Context, googleTaskID string) (bool, error) {
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
