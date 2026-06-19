package googletasks

import (
	"strings"
	"testing"
)

// Returns no sync state for a Google Task ID that has never been stored.
func TestGetSyncStateReturnsNotFoundForUnknownTask(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := newTestRepo(t)

	state, found, err := repo.GetSyncState(ctx, "google-1")
	if err != nil {
		t.Fatalf("get sync state: %v", err)
	}
	if found {
		t.Fatalf("expected no sync state, got %+v", state)
	}
}

// Returns an unfinalized sync state after a task mapping is saved.
func TestGetSyncStateReturnsUnfinalizedSavedTask(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := newTestRepo(t)
	params := syncedTaskParams()

	if err := repo.SaveSyncedTask(ctx, params); err != nil {
		t.Fatalf("save synced task: %v", err)
	}

	state, found, err := repo.GetSyncState(ctx, params.GoogleTaskID)
	if err != nil {
		t.Fatalf("get sync state: %v", err)
	}
	if !found {
		t.Fatal("expected sync state")
	}
	if state.GoogleTaskID != params.GoogleTaskID ||
		state.TickTickTaskID != params.TickTickTaskID ||
		state.PostSyncAction != params.PostSyncAction ||
		state.IsGoogleFinalized() {
		t.Fatalf("unexpected sync state: %+v", state)
	}
}

// Reports an error when sync state cannot be queried from the database.
func TestGetSyncStateReturnsErrorWhenDatabaseIsClosed(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := newTestRepo(t)
	if err := repo.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	state, found, err := repo.GetSyncState(ctx, "google-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if found {
		t.Fatalf("expected no sync state, got %+v", state)
	}
	if !strings.Contains(err.Error(), "get google task sync state") {
		t.Fatalf("unexpected error: %v", err)
	}
}
