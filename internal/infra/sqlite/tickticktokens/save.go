package tickticktokens

import (
	"context"
	"fmt"
	"time"
)

const queryUpsertToken = `
INSERT INTO ticktick_tokens (
  provider,
  access_token,
  token_type,
  scope,
  expires_at,
  refresh_token,
  updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider) DO UPDATE SET
  access_token = excluded.access_token,
  token_type = excluded.token_type,
  scope = excluded.scope,
  expires_at = excluded.expires_at,
  refresh_token = excluded.refresh_token,
  updated_at = excluded.updated_at;`

// Save stores the latest TickTick OAuth token.
func (r *Repo) Save(ctx context.Context, token Token) error {
	updatedAt := token.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, queryUpsertToken,
		providerTickTick,
		token.AccessToken,
		token.TokenType,
		token.Scope,
		formatTime(token.ExpiresAt),
		token.RefreshToken,
		formatTime(updatedAt),
	)
	if err != nil {
		return fmt.Errorf("save ticktick token: %w", err)
	}

	return nil
}
