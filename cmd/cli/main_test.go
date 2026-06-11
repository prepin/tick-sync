package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
)

func TestPrintTasksPrintsEmptyState(t *testing.T) {
	var out bytes.Buffer

	printTasks(&out, "@default", nil)

	got := out.String()
	if !strings.Contains(got, "Google task list: @default") {
		t.Fatalf("expected task list header, got %q", got)
	}
	if !strings.Contains(got, "No uncompleted tasks found.") {
		t.Fatalf("expected empty state, got %q", got)
	}
}

func TestPostSyncActionFromConfigDefaultsToComplete(t *testing.T) {
	got, err := postSyncActionFromConfig("")
	if err != nil {
		t.Fatalf("post sync action from config: %v", err)
	}
	if got != googletasksync.PostSyncActionComplete {
		t.Fatalf("unexpected action: %s", got)
	}
}

func TestPostSyncActionFromConfigParsesDelete(t *testing.T) {
	got, err := postSyncActionFromConfig("delete")
	if err != nil {
		t.Fatalf("post sync action from config: %v", err)
	}
	if got != googletasksync.PostSyncActionDelete {
		t.Fatalf("unexpected action: %s", got)
	}
}

func TestPostSyncActionFromConfigRejectsInvalidAction(t *testing.T) {
	_, err := postSyncActionFromConfig("archive")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "GOOGLE_POST_SYNC_ACTION") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrintSyncSummary(t *testing.T) {
	var out bytes.Buffer

	printSyncSummary(&out, googletasksync.SyncSummary{
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

func TestPrintTasksPrintsTaskFields(t *testing.T) {
	var out bytes.Buffer

	printTasks(&out, "@default", []googletasksync.GoogleTask{
		{
			ID:      "task-1",
			Title:   "Buy milk",
			Status:  "needsAction",
			Due:     "2026-06-12T00:00:00.000Z",
			Updated: "2026-06-10T10:00:00.000Z",
			Notes:   "Remember lactose-free",
		},
	})

	got := out.String()
	for _, want := range []string{
		"- id: task-1",
		"title: Buy milk",
		"status: needsAction",
		"due: 2026-06-12T00:00:00.000Z",
		"updated: 2026-06-10T10:00:00.000Z",
		"notes: |",
		"Remember lactose-free",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got %q", want, got)
		}
	}
}
