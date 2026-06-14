package googletasks

import (
	"context"
	"fmt"

	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
)

func (r *GoogleTasksRepo) SaveSyncedTask(ctx context.Context, record googletasksync.SyncedTaskRecord) error {
	_, err := r.db.ExecContext(ctx, queryInsertSyncedTask,
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
