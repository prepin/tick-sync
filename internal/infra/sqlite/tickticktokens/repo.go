package tickticktokens

import (
	"database/sql"
	"errors"
	"time"
)

const providerTickTick = "ticktick"

// ErrTokenNotFound reports that TickTick has not been connected yet.
var ErrTokenNotFound = errors.New("ticktick access token missing")

// Token is the persisted TickTick OAuth token.
type Token struct {
	AccessToken  string
	TokenType    string
	Scope        string
	ExpiresAt    time.Time
	RefreshToken string
	UpdatedAt    time.Time
}

// Repo stores TickTick OAuth tokens in SQLite.
type Repo struct {
	db *sql.DB
}

// New creates a Repo that uses the provided database.
func New(db *sql.DB) (*Repo, error) {
	if db == nil {
		return nil, errors.New("ticktick tokens repo: db is nil")
	}

	return &Repo{db: db}, nil
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}
