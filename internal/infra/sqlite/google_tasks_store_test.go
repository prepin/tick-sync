package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
	_ "modernc.org/sqlite"
)

// Creates the synced_google_tasks table in the database.
func TestNewGoogleTasksStoreCreatesTable(t *testing.T) {
	t.Parallel()
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

// Does not create a store when the database handle is nil.
func TestNewGoogleTasksStoreRejectsNilDB(t *testing.T) {
	t.Parallel()
	_, err := NewGoogleTasksStore(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// Does not create a store when the database is closed and table creation fails.
func TestNewGoogleTasksStoreReturnsErrorWhenTableCreationFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := openTestDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	_, err := NewGoogleTasksStore(ctx, db)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create synced_google_tasks table") {
		t.Fatalf("expected error about table creation, got %v", err)
	}
}

// Returns false for a Google Task ID that has never been stored.
func TestGoogleTasksStoreIsProcessedReturnsFalseForUnknownTask(t *testing.T) {
	t.Parallel()
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

// Reports an error when IsProcessed cannot query the database.
func TestGoogleTasksStoreIsProcessedReturnsErrorWhenDatabaseIsClosed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newTestStore(t, ctx)
	if err := store.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	processed, err := store.IsProcessed(ctx, "google-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if processed {
		t.Fatal("expected task to be unprocessed")
	}
}

// Records a synced task and returns true for subsequent IsProcessed checks.
func TestGoogleTasksStoreMarkProcessedRecordsTask(t *testing.T) {
	t.Parallel()
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

// Stores all record fields (updated, title, ticktick ID, action, synced at) in the database.
func TestGoogleTasksStoreMarkProcessedStoresRecordFields(t *testing.T) {
	t.Parallel()
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

// Does not insert duplicate rows when the same record is marked processed twice.
func TestGoogleTasksStoreMarkProcessedIsIdempotent(t *testing.T) {
	t.Parallel()
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

// Reports an error when MarkProcessed cannot write to the database.
func TestGoogleTasksStoreMarkProcessedReturnsErrorWhenDatabaseIsClosed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newTestStore(t, ctx)
	if err := store.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	err := store.MarkProcessed(ctx, syncedTaskRecord())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mark synced google task") {
		t.Fatalf("expected error about marking synced task, got %v", err)
	}
}

// Converts a time value to RFC3339Nano format in UTC regardless of the input timezone.
func TestFormatTimeReturnsRFC3339NanoUTC(t *testing.T) {
	t.Parallel()
	input := time.Date(2026, 6, 10, 12, 0, 0, 123, time.FixedZone("EST", -5*60*60))
	got := formatTime(input)
	if got != "2026-06-10T17:00:00.000000123Z" {
		t.Fatalf("unexpected formatted time: %q", got)
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
