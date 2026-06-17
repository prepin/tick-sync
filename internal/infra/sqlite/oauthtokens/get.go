package oauthtokens

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const queryGetToken = `
SELECT access_token, token_type, scope, expires_at, refresh_token, updated_at, refresh_reminder_task_id, refresh_reminder_created_at
FROM oauth_tokens
WHERE provider = ?;`

// Get returns the latest stored OAuth token for a provider.
func (r *Repo) Get(ctx context.Context, provider string) (Token, error) {
	var token Token
	var expiresAt string
	var updatedAt string
	var refreshReminderTaskID sql.NullString
	var refreshReminderCreatedAt sql.NullString
	err := r.db.QueryRowContext(ctx, queryGetToken, provider).Scan(
		&token.AccessToken,
		&token.TokenType,
		&token.Scope,
		&expiresAt,
		&token.RefreshToken,
		&updatedAt,
		&refreshReminderTaskID,
		&refreshReminderCreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Token{}, ErrTokenNotFound
	}
	if err != nil {
		return Token{}, fmt.Errorf("get oauth token %s: %w", provider, err)
	}

	token.ExpiresAt, err = parseOptionalTime(expiresAt)
	if err != nil {
		return Token{}, fmt.Errorf("parse oauth token expiry: %w", err)
	}
	token.UpdatedAt, err = parseOptionalTime(updatedAt)
	if err != nil {
		return Token{}, fmt.Errorf("parse oauth token update time: %w", err)
	}
	token.RefreshReminderTaskID = refreshReminderTaskID.String
	token.RefreshReminderCreatedAt, err = parseOptionalTime(refreshReminderCreatedAt.String)
	if err != nil {
		return Token{}, fmt.Errorf("parse oauth token refresh reminder creation time: %w", err)
	}

	return token, nil
}
