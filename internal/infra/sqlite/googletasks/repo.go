package googletasks

import (
	"database/sql"
	"errors"
	"time"
)

// Repo is the SQLite implementation of SyncedTaskRepository.
type Repo struct {
	db *sql.DB
}

// New creates a Repo that uses the provided database.
func New(db *sql.DB) (*Repo, error) {
	if db == nil {
		return nil, errors.New("google tasks repo: db is nil")
	}

	return &Repo{db: db}, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
