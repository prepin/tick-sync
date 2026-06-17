package tickticktokens

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const queryGetToken = `
SELECT access_token, token_type, scope, expires_at, refresh_token, updated_at
FROM ticktick_tokens
WHERE provider = ?;`

// Get returns the latest stored TickTick OAuth token.
func (r *Repo) Get(ctx context.Context) (Token, error) {
	var token Token
	var expiresAt string
	var updatedAt string
	err := r.db.QueryRowContext(ctx, queryGetToken, providerTickTick).Scan(
		&token.AccessToken,
		&token.TokenType,
		&token.Scope,
		&expiresAt,
		&token.RefreshToken,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Token{}, ErrTokenNotFound
	}
	if err != nil {
		return Token{}, fmt.Errorf("get ticktick token: %w", err)
	}

	token.ExpiresAt, err = parseOptionalTime(expiresAt)
	if err != nil {
		return Token{}, fmt.Errorf("parse ticktick token expiry: %w", err)
	}
	token.UpdatedAt, err = parseOptionalTime(updatedAt)
	if err != nil {
		return Token{}, fmt.Errorf("parse ticktick token update time: %w", err)
	}

	return token, nil
}

// GetAccessToken returns only the bearer token value required by the TickTick API client.
func (r *Repo) GetAccessToken(ctx context.Context) (string, error) {
	token, err := r.Get(ctx)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
