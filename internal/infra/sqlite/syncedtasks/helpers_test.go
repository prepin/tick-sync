package syncedtasks

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/prepin/tick-sync/internal/application/googletasksync"
	_ "modernc.org/sqlite"
)

// Creates a Repo with a fresh in-memory SQLite database for testing.
func newTestRepo(t *testing.T, ctx context.Context) *Repo {
	t.Helper()

	repo, err := New(ctx, openTestDB(t))
	if err != nil {
		t.Fatalf("new synced tasks repo: %v", err)
	}

	return repo
}

// Opens a temporary SQLite database that is cleaned up after the test completes.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "tick-sync.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close sqlite db: %v", err)
		}
	})

	return db
}

// Returns a default SaveSyncedTaskParams fixture for a single google-1 task synced to ticktick-1 with complete action.
func syncedTaskParams() googletasksync.SaveSyncedTaskParams {
	return googletasksync.SaveSyncedTaskParams{
		GoogleTaskID:   "google-1",
		GoogleUpdated:  "2026-06-10T10:00:00.000Z",
		GoogleTitle:    "Buy milk",
		TickTickTaskID: "ticktick-1",
		PostSyncAction: googletasksync.PostSyncActionComplete,
		SyncedAt:       time.Date(2026, 6, 10, 12, 0, 0, 123, time.UTC),
	}
}
