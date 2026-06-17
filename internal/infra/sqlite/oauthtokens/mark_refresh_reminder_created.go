package oauthtokens

import (
	"context"
	"fmt"
	"time"
)

const queryMarkRefreshReminderCreated = `
UPDATE oauth_tokens
SET refresh_reminder_task_id = ?, refresh_reminder_created_at = ?
WHERE provider = ? AND access_token = ?;`

// MarkRefreshReminderCreated records that a reminder task has already been created for the current provider token.
func (r *Repo) MarkRefreshReminderCreated(ctx context.Context, provider string, accessToken string, taskID string, createdAt time.Time) error {
	result, err := r.db.ExecContext(ctx, queryMarkRefreshReminderCreated,
		taskID,
		formatTime(createdAt),
		provider,
		accessToken,
	)
	if err != nil {
		return fmt.Errorf("mark oauth refresh reminder created: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count oauth refresh reminder marker rows: %w", err)
	}
	if affected == 0 {
		return ErrTokenNotFound
	}
	return nil
}
