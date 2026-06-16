package googletasks

import (
	"strings"
	"testing"
)

// Records a synced task and returns true for subsequent IsProcessed checks.
func TestSaveSyncedTaskRecordsTask(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := newTestRepo(t, ctx)

	if err := repo.SaveSyncedTask(ctx, syncedTaskParams()); err != nil {
		t.Fatalf("save synced task: %v", err)
	}

	processed, err := repo.IsProcessed(ctx, "google-1")
	if err != nil {
		t.Fatalf("is processed: %v", err)
	}
	if !processed {
		t.Fatal("expected task to be processed")
	}
}

// Stores all record fields (updated, title, ticktick ID, action, synced at) in the database.
func TestSaveSyncedTaskStoresRecordFields(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := openTestDB(t)
	repo, err := New(ctx, db)
	if err != nil {
		t.Fatalf("new google tasks repo: %v", err)
	}

	params := syncedTaskParams()
	if err := repo.SaveSyncedTask(ctx, params); err != nil {
		t.Fatalf("save synced task: %v", err)
	}

	var got struct {
		GoogleUpdated  string
		GoogleTitle    string
		TickTickTaskID string
		PostSyncAction string
		SyncedAt       string
	}
	err = db.QueryRowContext(ctx, `
SELECT google_updated, google_title, ticktick_task_id, post_sync_action, synced_at
FROM synced_google_tasks
WHERE google_task_id = ?;`, params.GoogleTaskID).Scan(
		&got.GoogleUpdated,
		&got.GoogleTitle,
		&got.TickTickTaskID,
		&got.PostSyncAction,
		&got.SyncedAt,
	)
	if err != nil {
		t.Fatalf("query stored record: %v", err)
	}

	if got.GoogleUpdated != params.GoogleUpdated ||
		got.GoogleTitle != params.GoogleTitle ||
		got.TickTickTaskID != params.TickTickTaskID ||
		got.PostSyncAction != string(params.PostSyncAction) ||
		got.SyncedAt != formatTime(params.SyncedAt) {
		t.Fatalf("unexpected stored record: %+v", got)
	}
}

// Does not insert duplicate rows when the same record is saved twice.
func TestSaveSyncedTaskIsIdempotent(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := openTestDB(t)
	repo, err := New(ctx, db)
	if err != nil {
		t.Fatalf("new google tasks repo: %v", err)
	}

	params := syncedTaskParams()
	if err := repo.SaveSyncedTask(ctx, params); err != nil {
		t.Fatalf("save synced task: %v", err)
	}
	if err := repo.SaveSyncedTask(ctx, params); err != nil {
		t.Fatalf("save synced task again: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM synced_google_tasks WHERE google_task_id = ?;`, params.GoogleTaskID).
		Scan(&count); err != nil {
		t.Fatalf("count records: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one record, got %d", count)
	}
}

// Reports an error when SaveSyncedTask cannot write to the database.
func TestSaveSyncedTaskReturnsErrorWhenDatabaseIsClosed(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := newTestRepo(t, ctx)
	if err := repo.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	err := repo.SaveSyncedTask(ctx, syncedTaskParams())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "save synced google task") {
		t.Fatalf("expected error about saving synced task, got %v", err)
	}
}
