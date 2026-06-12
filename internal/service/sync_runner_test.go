package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
	_ "modernc.org/sqlite"
)

// Does not create a runner when the configured post-sync action is invalid.
func TestNewSyncRunnerRejectsInvalidPostSyncAction(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{
		GooglePostSyncAction: "archive",
	}

	_, _, err := NewSyncRunner(ctx, cfg)
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

	_, _, err := NewSyncRunner(ctx, cfg)
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

	_, _, err := NewSyncRunner(ctx, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create ticktick client") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Syncs a single Google Task to TickTick, completes the source task, and logs the summary.
func TestRunOnceSyncsTaskAndLogsSummary(t *testing.T) {
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

	buf := captureSlogOutput(t)
	runner, cleanup, err := NewSyncRunner(ctx, cfg)
	if err != nil {
		t.Fatalf("new sync runner: %v", err)
	}
	t.Cleanup(func() { _ = cleanup() })

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatalf("run once: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `seen=1`) {
		t.Fatalf("expected seen=1 in summary, got %q", got)
	}
	if !strings.Contains(got, `created=1`) {
		t.Fatalf("expected created=1 in summary, got %q", got)
	}
	if !strings.Contains(got, `completed=1`) {
		t.Fatalf("expected completed=1 in summary, got %q", got)
	}
}

// Propagates the sync error to the caller when the usecase fails.
func TestRunOnceReportsSyncError(t *testing.T) {
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

	buf := captureSlogOutput(t)
	runner, cleanup, err := NewSyncRunner(ctx, cfg)
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
	if !strings.Contains(buf.String(), "level=ERROR") {
		t.Fatalf("expected error level log, got %q", buf.String())
	}
}

// Returns "complete" as the default post-sync action when the config value is empty.
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

// Returns "delete" as the post-sync action when configured.
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

// Reports an error when the post-sync action value is not "complete" or "delete".
func TestPostSyncActionFromConfigReportsErrorForInvalidAction(t *testing.T) {
	t.Parallel()
	_, err := PostSyncActionFromConfig("archive")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "GOOGLE_POST_SYNC_ACTION") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Logs a single summary line with all field values and INFO level when there are no errors.
func TestPrintSyncSummaryLogsFieldValuesAtInfoLevel(t *testing.T) {
	buf := captureSlogOutput(t)

	PrintSyncSummary(googletasksync.SyncSummary{
		Seen:      4,
		Created:   3,
		Skipped:   1,
		Failed:    0,
		Completed: 3,
		Deleted:   0,
	})

	got := buf.String()
	if !strings.Contains(got, "level=INFO") {
		t.Fatalf("expected info level log, got %q", got)
	}
	if !strings.Contains(got, `seen=4`) {
		t.Fatalf("expected seen=4 in log, got %q", got)
	}
	if !strings.Contains(got, `created=3`) {
		t.Fatalf("expected created=3 in log, got %q", got)
	}
	if !strings.Contains(got, `skipped=1`) {
		t.Fatalf("expected skipped=1 in log, got %q", got)
	}
	if !strings.Contains(got, `failed=0`) {
		t.Fatalf("expected failed=0 in log, got %q", got)
	}
	if !strings.Contains(got, `completed=3`) {
		t.Fatalf("expected completed=3 in log, got %q", got)
	}
	if !strings.Contains(got, `deleted=0`) {
		t.Fatalf("expected deleted=0 in log, got %q", got)
	}
}

// Logs the error level with an errors field containing non-nil error messages when the sync encountered failures.
func TestPrintSyncSummaryLogsErrorsAtErrorLevel(t *testing.T) {
	buf := captureSlogOutput(t)

	PrintSyncSummary(googletasksync.SyncSummary{
		Seen:      1,
		Created:   0,
		Skipped:   0,
		Failed:    1,
		Completed: 0,
		Deleted:   0,
		Errors:    []error{errors.New("ticktick unavailable"), nil, errors.New("db write failed")},
	})

	got := buf.String()
	if !strings.Contains(got, "level=ERROR") {
		t.Fatalf("expected error level log, got %q", got)
	}
	if !strings.Contains(got, `errors="ticktick unavailable, db write failed"`) {
		t.Fatalf("expected errors in log, got %q", got)
	}
}

// Creates HTTP test servers for Google Tasks and TickTick APIs
// that respond to a one-task sync scenario (list, create, complete).
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

func captureSlogOutput(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return &buf
}
