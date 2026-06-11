package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
	_ "modernc.org/sqlite"
)

func TestNewGoogleTasksStoreCreatesTable(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	_, err := NewGoogleTasksStore(ctx, db)
	if err != nil {
		t.Fatalf("new google tasks store: %v", err)
	}

	var tableName string
	err = db.QueryRowContext(ctx, `
SELECT name
FROM sqlite_master
WHERE type = 'table' AND name = 'synced_google_tasks';`).Scan(&tableName)
	if err != nil {
		t.Fatalf("query table: %v", err)
	}
	if tableName != "synced_google_tasks" {
		t.Fatalf("unexpected table name: %q", tableName)
	}
}

func TestGoogleTasksStoreIsProcessedReturnsFalseForUnknownTask(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, ctx)

	processed, err := store.IsProcessed(ctx, "google-1")
	if err != nil {
		t.Fatalf("is processed: %v", err)
	}
	if processed {
		t.Fatal("expected task to be unprocessed")
	}
}

func TestGoogleTasksStoreMarkProcessedRecordsTask(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, ctx)

	if err := store.MarkProcessed(ctx, syncedTaskRecord()); err != nil {
		t.Fatalf("mark processed: %v", err)
	}

	processed, err := store.IsProcessed(ctx, "google-1")
	if err != nil {
		t.Fatalf("is processed: %v", err)
	}
	if !processed {
		t.Fatal("expected task to be processed")
	}
}

func TestGoogleTasksStoreMarkProcessedStoresRecordFields(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	store, err := NewGoogleTasksStore(ctx, db)
	if err != nil {
		t.Fatalf("new google tasks store: %v", err)
	}

	record := syncedTaskRecord()
	if err := store.MarkProcessed(ctx, record); err != nil {
		t.Fatalf("mark processed: %v", err)
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
WHERE google_task_id = ?;`, record.GoogleTaskID).Scan(
		&got.GoogleUpdated,
		&got.GoogleTitle,
		&got.TickTickTaskID,
		&got.PostSyncAction,
		&got.SyncedAt,
	)
	if err != nil {
		t.Fatalf("query stored record: %v", err)
	}

	if got.GoogleUpdated != record.GoogleUpdated ||
		got.GoogleTitle != record.GoogleTitle ||
		got.TickTickTaskID != record.TickTickTaskID ||
		got.PostSyncAction != string(record.PostSyncAction) ||
		got.SyncedAt != formatTime(record.SyncedAt) {
		t.Fatalf("unexpected stored record: %+v", got)
	}
}

func TestGoogleTasksStoreMarkProcessedIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	store, err := NewGoogleTasksStore(ctx, db)
	if err != nil {
		t.Fatalf("new google tasks store: %v", err)
	}

	record := syncedTaskRecord()
	if err := store.MarkProcessed(ctx, record); err != nil {
		t.Fatalf("mark processed: %v", err)
	}
	if err := store.MarkProcessed(ctx, record); err != nil {
		t.Fatalf("mark processed again: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM synced_google_tasks WHERE google_task_id = ?;`, record.GoogleTaskID).Scan(&count); err != nil {
		t.Fatalf("count records: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one record, got %d", count)
	}
}

func TestNewGoogleTasksStoreRejectsNilDB(t *testing.T) {
	_, err := NewGoogleTasksStore(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func newTestStore(t *testing.T, ctx context.Context) *GoogleTasksStore {
	t.Helper()

	store, err := NewGoogleTasksStore(ctx, openTestDB(t))
	if err != nil {
		t.Fatalf("new google tasks store: %v", err)
	}

	return store
}

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

func syncedTaskRecord() googletasksync.SyncedTaskRecord {
	return googletasksync.SyncedTaskRecord{
		GoogleTaskID:   "google-1",
		GoogleUpdated:  "2026-06-10T10:00:00.000Z",
		GoogleTitle:    "Buy milk",
		TickTickTaskID: "ticktick-1",
		PostSyncAction: googletasksync.PostSyncActionComplete,
		SyncedAt:       time.Date(2026, 6, 10, 12, 0, 0, 123, time.UTC),
	}
}
