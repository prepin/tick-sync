package googletasks

import (
	"testing"
)

// Allows creating a repo when the database handle is valid.
func TestNewCreatesRepo(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	db := openTestDB(t)

	_, err := New(ctx, db)
	if err != nil {
		t.Fatalf("new google tasks repo: %v", err)
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
