package oauthtokens

import (
	"context"
	"fmt"
	"time"
)

//nolint:gosec // SQL column names contain token fields; no credential values are hardcoded.
const queryUpsertToken = `
INSERT INTO oauth_tokens (
  provider,
  access_token,
  token_type,
  scope,
  expires_at,
  refresh_token,
  updated_at,
  refresh_reminder_task_id,
  refresh_reminder_created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider) DO UPDATE SET
  access_token = excluded.access_token,
  token_type = excluded.token_type,
  scope = excluded.scope,
  expires_at = excluded.expires_at,
  refresh_token = excluded.refresh_token,
  updated_at = excluded.updated_at,
  refresh_reminder_task_id = CASE
    WHEN oauth_tokens.access_token = excluded.access_token THEN oauth_tokens.refresh_reminder_task_id
    ELSE excluded.refresh_reminder_task_id
  END,
  refresh_reminder_created_at = CASE
    WHEN oauth_tokens.access_token = excluded.access_token THEN oauth_tokens.refresh_reminder_created_at
    ELSE excluded.refresh_reminder_created_at
  END;`

// Save stores the latest OAuth token for a provider.
func (r *Repo) Save(ctx context.Context, provider string, token Token) error {
	updatedAt := token.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, queryUpsertToken,
		provider,
		token.AccessToken,
		token.TokenType,
		token.Scope,
		formatTime(token.ExpiresAt),
		token.RefreshToken,
		formatTime(updatedAt),
		token.RefreshReminderTaskID,
		formatTime(token.RefreshReminderCreatedAt),
	)
	if err != nil {
		return fmt.Errorf("save oauth token %s: %w", provider, err)
	}

	return nil
}
