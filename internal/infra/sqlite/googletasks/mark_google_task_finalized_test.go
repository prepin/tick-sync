package googletasks

import (
	"strings"
	"testing"
	"time"
)

// Records the time when a synced Google task was completed or deleted.
func TestMarkGoogleTaskFinalizedStoresFinalizationTime(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := newTestRepo(t)
	params := syncedTaskParams()
	finalizedAt := time.Date(2026, 6, 10, 13, 0, 0, 0, time.UTC)

	if err := repo.SaveSyncedTask(ctx, params); err != nil {
		t.Fatalf("save synced task: %v", err)
	}
	if err := repo.MarkGoogleTaskFinalized(ctx, params.GoogleTaskID, finalizedAt); err != nil {
		t.Fatalf("mark google task finalized: %v", err)
	}

	state, found, err := repo.GetSyncState(ctx, params.GoogleTaskID)
	if err != nil {
		t.Fatalf("get sync state: %v", err)
	}
	if !found {
		t.Fatal("expected sync state")
	}
	if !state.GoogleFinalizedAt.Equal(finalizedAt) {
		t.Fatalf("unexpected finalization time: %s", state.GoogleFinalizedAt)
	}
}

// Reports an error when the finalization marker cannot be written.
func TestMarkGoogleTaskFinalizedReturnsErrorWhenDatabaseIsClosed(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := newTestRepo(t)
	if err := repo.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	err := repo.MarkGoogleTaskFinalized(ctx, "google-1", time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mark google task") {
		t.Fatalf("unexpected error: %v", err)
	}
}
