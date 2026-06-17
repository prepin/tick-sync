package tickticktokens

import "testing"

// Allows creating a repo when the database handle is valid.
func TestNewCreatesRepo(t *testing.T) {
	t.Parallel()

	_, err := New(openTestDB(t))
	if err != nil {
		t.Fatalf("new ticktick token repo: %v", err)
	}
}

// Does not create a repo when the database handle is nil.
func TestNewRejectsNilDB(t *testing.T) {
	t.Parallel()

	_, err := New(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
