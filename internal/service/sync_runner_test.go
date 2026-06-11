package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
	_ "modernc.org/sqlite"
)

func TestPostSyncActionFromConfigDefaultsToComplete(t *testing.T) {
	t.Parallel()
	got, err := PostSyncActionFromConfig("")
	if err != nil {
		t.Fatalf("post sync action from config: %v", err)
	}
	if got != googletasksync.PostSyncActionComplete {
		t.Fatalf("unexpected action: %s", got)
	}
}

func TestPostSyncActionFromConfigParsesDelete(t *testing.T) {
	t.Parallel()
	got, err := PostSyncActionFromConfig("delete")
	if err != nil {
		t.Fatalf("post sync action from config: %v", err)
	}
	if got != googletasksync.PostSyncActionDelete {
		t.Fatalf("unexpected action: %s", got)
	}
}

func TestPostSyncActionFromConfigRejectsInvalidAction(t *testing.T) {
	t.Parallel()
	_, err := PostSyncActionFromConfig("archive")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "GOOGLE_POST_SYNC_ACTION") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Prints all summary field values including zero counts.
func TestPrintSyncSummaryPrintsFieldValues(t *testing.T) {
	t.Parallel()
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

// Prints errors section with each non-nil error, skipping nil entries.
func TestPrintSyncSummaryPrintsErrorsWhenPresent(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer

	PrintSyncSummary(&out, googletasksync.SyncSummary{
		Seen:      1,
		Created:   0,
		Skipped:   0,
		Failed:    1,
		Completed: 0,
		Deleted:   0,
		Errors:    []error{errors.New("ticktick unavailable"), nil, errors.New("db write failed")},
	})

	got := out.String()
	if !strings.Contains(got, "Errors:") {
		t.Fatalf("expected output to contain Errors header, got %q", got)
	}
	if !strings.Contains(got, "- ticktick unavailable") {
		t.Fatalf("expected first error in output, got %q", got)
	}
	if !strings.Contains(got, "- db write failed") {
		t.Fatalf("expected second error in output, got %q", got)
	}
	if strings.Contains(got, "<nil>") {
		t.Fatalf("did not expect nil error in output, got %q", got)
	}
}

// Does not create a runner when the configured post-sync action is invalid.
func TestNewSyncRunnerRejectsInvalidPostSyncAction(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{
		GooglePostSyncAction: "archive",
	}

	_, _, err := NewSyncRunner(ctx, cfg, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "GOOGLE_POST_SYNC_ACTION") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Does not create a runner when the database path is unwritable.
func TestNewSyncRunnerRejectsDBOpenFailure(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	cfg := config.Config{
		DBPath:              dir,
		TickTickAccessToken: "test-token",
	}

	_, _, err := NewSyncRunner(ctx, cfg, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create google tasks store") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Does not create a runner when the TickTick access token is missing.
func TestNewSyncRunnerRejectsMissingTickTickAccessToken(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")
	cfg := config.Config{
		DBPath:              dbPath,
		GoogleAPIEndpoint:   "https://example.com/",
		TickTickAccessToken: "",
	}

	_, _, err := NewSyncRunner(ctx, cfg, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create ticktick client") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Syncs a single Google Task to TickTick, completes the source task, and prints the summary.
func TestRunOnceSyncsTaskAndPrintsSummary(t *testing.T) {
	ctx := context.Background()
	googleServer, ticktickServer := startMockServers(t)
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")

	cfg := config.Config{
		DBPath:              dbPath,
		GoogleAPIEndpoint:   googleServer.URL + "/",
		GoogleTaskListID:    "@default",
		TickTickAPIBaseURL:  ticktickServer.URL,
		TickTickAccessToken: "test-token",
	}

	var out bytes.Buffer
	runner, cleanup, err := NewSyncRunner(ctx, cfg, &out)
	if err != nil {
		t.Fatalf("new sync runner: %v", err)
	}
	t.Cleanup(func() { _ = cleanup() })

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatalf("run once: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Seen: 1") {
		t.Fatalf("expected Seen: 1 in summary, got %q", got)
	}
	if !strings.Contains(got, "Created: 1") {
		t.Fatalf("expected Created: 1 in summary, got %q", got)
	}
	if !strings.Contains(got, "Completed: 1") {
		t.Fatalf("expected Completed: 1 in summary, got %q", got)
	}
}

// Propagates the sync error to the caller when the usecase fails.
func TestRunOnceReportsSyncError(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")

	// A TickTick server that always returns an error to trigger usecase failure.
	ticktickServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(ticktickServer.Close)

	// Google server that returns a task so the usecase tries to create in TickTick.
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

	var out bytes.Buffer
	runner, cleanup, err := NewSyncRunner(ctx, cfg, &out)
	if err != nil {
		t.Fatalf("new sync runner: %v", err)
	}
	t.Cleanup(func() { _ = cleanup() })

	err = runner.RunOnce(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "sync google tasks to ticktick") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// startMockServers creates httptest servers for Google Tasks and TickTick APIs
// that respond to a simple one-task sync scenario.
func startMockServers(t *testing.T) (googleServer, ticktickServer *httptest.Server) {
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
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id":     "g1",
				"status": "completed",
			})
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
