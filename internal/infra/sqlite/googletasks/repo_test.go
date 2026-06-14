package googletasks

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
func TestNewGoogleTasksRepoCreatesTable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := openTestDB(t)

	_, err := NewGoogleTasksRepo(ctx, db)
	if err != nil {
		t.Fatalf("new google tasks repo: %v", err)
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

// Does not create a repo when the database handle is nil.
func TestNewGoogleTasksRepoRejectsNilDB(t *testing.T) {
	t.Parallel()
	_, err := NewGoogleTasksRepo(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// Does not create a repo when the database is closed and table creation fails.
func TestNewGoogleTasksRepoReturnsErrorWhenTableCreationFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := openTestDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	_, err := NewGoogleTasksRepo(ctx, db)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create synced_google_tasks table") {
		t.Fatalf("expected error about table creation, got %v", err)
	}
}

// Returns false for a Google Task ID that has never been stored.
func TestGoogleTasksRepoIsProcessedReturnsFalseForUnknownTask(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newTestRepo(t, ctx)

	processed, err := repo.IsProcessed(ctx, "google-1")
	if err != nil {
		t.Fatalf("is processed: %v", err)
	}
	if processed {
		t.Fatal("expected task to be unprocessed")
	}
}

// Reports an error when IsProcessed cannot query the database.
func TestGoogleTasksRepoIsProcessedReturnsErrorWhenDatabaseIsClosed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newTestRepo(t, ctx)
	if err := repo.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	processed, err := repo.IsProcessed(ctx, "google-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if processed {
		t.Fatal("expected task to be unprocessed")
	}
}

// Records a synced task and returns true for subsequent IsProcessed checks.
func TestGoogleTasksRepoSaveSyncedTaskRecordsTask(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newTestRepo(t, ctx)

	if err := repo.SaveSyncedTask(ctx, syncedTaskRecord()); err != nil {
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
func TestGoogleTasksRepoSaveSyncedTaskStoresRecordFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := openTestDB(t)
	repo, err := NewGoogleTasksRepo(ctx, db)
	if err != nil {
		t.Fatalf("new google tasks repo: %v", err)
	}

	record := syncedTaskRecord()
	if err := repo.SaveSyncedTask(ctx, record); err != nil {
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

// Does not insert duplicate rows when the same record is saved twice.
func TestGoogleTasksRepoSaveSyncedTaskIsIdempotent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := openTestDB(t)
	repo, err := NewGoogleTasksRepo(ctx, db)
	if err != nil {
		t.Fatalf("new google tasks repo: %v", err)
	}

	record := syncedTaskRecord()
	if err := repo.SaveSyncedTask(ctx, record); err != nil {
		t.Fatalf("save synced task: %v", err)
	}
	if err := repo.SaveSyncedTask(ctx, record); err != nil {
		t.Fatalf("save synced task again: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM synced_google_tasks WHERE google_task_id = ?;`, record.GoogleTaskID).Scan(&count); err != nil {
		t.Fatalf("count records: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one record, got %d", count)
	}
}

// Reports an error when SaveSyncedTask cannot write to the database.
func TestGoogleTasksRepoSaveSyncedTaskReturnsErrorWhenDatabaseIsClosed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newTestRepo(t, ctx)
	if err := repo.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	err := repo.SaveSyncedTask(ctx, syncedTaskRecord())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "save synced google task") {
		t.Fatalf("expected error about saving synced task, got %v", err)
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

// Creates a GoogleTasksRepo with a fresh in-memory SQLite database for testing.
func newTestRepo(t *testing.T, ctx context.Context) *GoogleTasksRepo {
	t.Helper()

	repo, err := NewGoogleTasksRepo(ctx, openTestDB(t))
	if err != nil {
		t.Fatalf("new google tasks repo: %v", err)
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

// Returns a default SyncedTaskRecord fixture for a single google-1 task synced to ticktick-1 with complete action.
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
