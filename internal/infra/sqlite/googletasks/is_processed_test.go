package googletasks

import (
	"testing"
)

// Returns false for a Google Task ID that has never been stored.
func TestIsProcessedReturnsFalseForUnknownTask(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := newTestRepo(t)

	processed, err := repo.IsProcessed(ctx, "google-1")
	if err != nil {
		t.Fatalf("is processed: %v", err)
	}
	if processed {
		t.Fatal("expected task to be unprocessed")
	}
}

// Reports an error when IsProcessed cannot query the database.
func TestIsProcessedReturnsErrorWhenDatabaseIsClosed(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := newTestRepo(t)
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
