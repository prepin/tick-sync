package oauthtokens

import (
	"context"
	"fmt"
	"time"
)

const queryMarkExpiryReminderCreated = `
UPDATE oauth_tokens
SET expiry_reminder_task_id = ?, expiry_reminder_created_at = ?
WHERE provider = ? AND access_token = ?;`

// MarkExpiryReminderCreated records the Google reminder for the current provider token.
func (r *Repo) MarkExpiryReminderCreated(
	ctx context.Context,
	provider string,
	accessToken string,
	taskID string,
	createdAt time.Time,
) error {
	result, err := r.db.ExecContext(ctx, queryMarkExpiryReminderCreated,
		taskID, formatTime(createdAt), provider, accessToken,
	)
	if err != nil {
		return fmt.Errorf("mark oauth expiry reminder created: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count oauth expiry reminder marker rows: %w", err)
	}
	if affected == 0 {
		return ErrTokenNotFound
	}
	return nil
}
