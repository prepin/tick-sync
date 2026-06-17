package oauthtokens

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/prepin/tick-sync/internal/infra/sqlite/migrate"
	_ "modernc.org/sqlite"
)

// Creates a Repo with a fresh SQLite database for testing.
func newTestRepo(t *testing.T) *Repo {
	t.Helper()

	repo, err := New(openTestDB(t))
	if err != nil {
		t.Fatalf("new oauth token repo: %v", err)
	}
	return repo
}

// Opens a temporary SQLite database with migrations applied.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "tick-sync.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := migrate.Up(t.Context(), db); err != nil {
		db.Close()
		t.Fatalf("run sqlite migrations: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close sqlite db: %v", err)
		}
	})
	return db
}

// Returns a default OAuth token fixture.
func tokenFixture() Token {
	return Token{
		AccessToken:  "access-1",
		TokenType:    "bearer",
		Scope:        "tasks:read tasks:write",
		ExpiresAt:    time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC),
		RefreshToken: "refresh-1",
		UpdatedAt:    time.Date(2026, 6, 17, 11, 0, 0, 0, time.UTC),
	}
}
