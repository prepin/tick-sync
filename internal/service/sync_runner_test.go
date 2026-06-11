package service

import (
	"bytes"
	"strings"
	"testing"

	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
)

func TestPostSyncActionFromConfigDefaultsToComplete(t *testing.T) {
	got, err := PostSyncActionFromConfig("")
	if err != nil {
		t.Fatalf("post sync action from config: %v", err)
	}
	if got != googletasksync.PostSyncActionComplete {
		t.Fatalf("unexpected action: %s", got)
	}
}

func TestPostSyncActionFromConfigParsesDelete(t *testing.T) {
	got, err := PostSyncActionFromConfig("delete")
	if err != nil {
		t.Fatalf("post sync action from config: %v", err)
	}
	if got != googletasksync.PostSyncActionDelete {
		t.Fatalf("unexpected action: %s", got)
	}
}

func TestPostSyncActionFromConfigRejectsInvalidAction(t *testing.T) {
	_, err := PostSyncActionFromConfig("archive")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "GOOGLE_POST_SYNC_ACTION") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrintSyncSummary(t *testing.T) {
	var out bytes.Buffer

	PrintSyncSummary(&out, googletasksync.SyncSummary{
		Seen:      4,
		Created:   3,
		Skipped:   1,
		Failed:    0,
		Completed: 3,
		Deleted:   0,
	})

	got := out.String()
	for _, want := range []string{
		"Sync summary:",
		"Seen: 4",
		"Created: 3",
		"Skipped: 1",
		"Failed: 0",
		"Completed: 3",
		"Deleted: 0",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got %q", want, got)
		}
	}
}
