package syncedtasks

import (
	"strings"
	"testing"
)

// Creates the synced_google_tasks table in the database.
func TestNewCreatesTable(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := openTestDB(t)

	_, err := New(ctx, db)
	if err != nil {
		t.Fatalf("new synced tasks repo: %v", err)
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
func TestNewRejectsNilDB(t *testing.T) {
	t.Parallel()
	_, err := New(t.Context(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// Does not create a repo when the database is closed and table creation fails.
func TestNewReturnsErrorWhenTableCreationFails(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := openTestDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	_, err := New(ctx, db)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create synced_google_tasks table") {
		t.Fatalf("expected error about table creation, got %v", err)
	}
}
