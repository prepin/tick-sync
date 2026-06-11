package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
)

func TestPrintTasksPrintsEmptyState(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

// Shows "notes:" label without pipe or indented content when task has no notes.
func TestPrintTasksPrintsNotesLabelWithoutContentWhenEmpty(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer

	printTasks(&out, "@default", []googletasksync.GoogleTask{
		{
			ID:    "task-1",
			Title: "Buy milk",
			Notes: "",
		},
	})

	got := out.String()
	if !strings.Contains(got, "notes:\n") {
		t.Fatalf("expected notes label on its own line, got %q", got)
	}
	if strings.Contains(got, "notes: |") {
		t.Fatalf("did not expect pipe for empty notes, got %q", got)
	}
}

// Lists uncompleted Google Tasks and prints their fields to the output writer.
func TestRunListPrintsTasksFromGoogleAPI(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]string{
				{"id": "g1", "title": "Buy milk", "status": "needsAction", "due": "2026-06-12T00:00:00.000Z", "updated": "2026-06-10T10:00:00.000Z"},
			},
		})
	}))
	t.Cleanup(server.Close)

	cfg := config.Config{
		GoogleAPIEndpoint: server.URL + "/",
		GoogleTaskListID:  "@default",
	}

	var out bytes.Buffer
	if err := runList(ctx, cfg, &out); err != nil {
		t.Fatalf("run list: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Google task list: @default") {
		t.Fatalf("expected task list header, got %q", got)
	}
	if !strings.Contains(got, "title: Buy milk") {
		t.Fatalf("expected task title in output, got %q", got)
	}
}

// Returns an error when the Google Tasks API responds with a non-2xx status.
func TestRunListReturnsErrorOnGoogleAPIFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	cfg := config.Config{
		GoogleAPIEndpoint: server.URL + "/",
		GoogleTaskListID:  "@default",
	}

	err := runList(ctx, cfg, io.Discard)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "list google tasks") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Runs a full sync cycle through the Google and TickTick mock APIs and reports success.
func TestRunSyncCompletesSyncCycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	googleServer, ticktickServer := startCLIMockServers(t)
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")

	cfg := config.Config{
		DBPath:              dbPath,
		GoogleAPIEndpoint:   googleServer.URL + "/",
		GoogleTaskListID:    "@default",
		TickTickAPIBaseURL:  ticktickServer.URL,
		TickTickAccessToken: "test-token",
	}

	if err := runSync(ctx, cfg); err != nil {
		t.Fatalf("run sync: %v", err)
	}
}

// Propagates the sync error when the TickTick API is unavailable.
func TestRunSyncReturnsErrorOnTickTickAPIFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")

	ticktickServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(ticktickServer.Close)

	googleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]string{
				{"id": "g1", "title": "Buy milk"},
			},
		})
	}))
	t.Cleanup(googleServer.Close)

	cfg := config.Config{
		DBPath:              dbPath,
		GoogleAPIEndpoint:   googleServer.URL + "/",
		GoogleTaskListID:    "@default",
		TickTickAPIBaseURL:  ticktickServer.URL,
		TickTickAccessToken: "test-token",
	}

	err := runSync(ctx, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "sync google tasks to ticktick") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// startCLIMockServers creates httptest servers for Google Tasks and TickTick APIs
// that respond to a simple one-task sync scenario.
func startCLIMockServers(t *testing.T) (googleServer, ticktickServer *httptest.Server) {
	t.Helper()

	googleServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]string{
					{"id": "g1", "title": "Buy milk"},
				},
			})
		case http.MethodPatch:
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "g1", "status": "completed"})
		default:
			t.Errorf("unexpected Google method: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(googleServer.Close)

	ticktickServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "t1"})
	}))
	t.Cleanup(ticktickServer.Close)

	return googleServer, ticktickServer
}
