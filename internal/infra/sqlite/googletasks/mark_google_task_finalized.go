package googletasks

import (
	"context"
	"fmt"
	"time"
)

const queryMarkGoogleTaskFinalized = `
UPDATE synced_google_tasks
SET google_finalized_at = ?
WHERE google_task_id = ?;`

// MarkGoogleTaskFinalized records that the source Google task was completed or deleted.
func (r *Repo) MarkGoogleTaskFinalized(ctx context.Context, googleTaskID string, finalizedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, queryMarkGoogleTaskFinalized, formatTime(finalizedAt), googleTaskID)
	if err != nil {
		return fmt.Errorf("mark google task %s finalized: %w", googleTaskID, err)
	}

	return nil
}
