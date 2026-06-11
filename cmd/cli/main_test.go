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
