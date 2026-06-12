package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prepin/tick-sync/internal/config"
)

// Does not create an app when the post-sync action is not "complete" or "delete".
func TestNewRejectsInvalidPostSyncAction(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{
		GooglePostSyncAction: "archive",
	}

	_, err := New(ctx, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "GOOGLE_POST_SYNC_ACTION") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Does not create an app when the database path is a directory or unwritable.
func TestNewRejectsDBOpenFailure(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	cfg := config.Config{
		DBPath:              dir,
		TickTickAccessToken: "test-token",
	}

	_, err := New(ctx, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create google tasks store") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Does not create an app when the TickTick access token is missing.
func TestNewRejectsMissingTickTickAccessToken(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")
	cfg := config.Config{
		DBPath:              dbPath,
		GoogleAPIEndpoint:   "https://example.com/",
		TickTickAccessToken: "",
	}

	_, err := New(ctx, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create ticktick client") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Runs the sync job once and returns nil when the context is cancelled after the first execution.
func TestAppRunStopsOnContextCancel(t *testing.T) {
	googleServer, ticktickServer := startMockServers(t)
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")

	cfg := config.Config{
		DBPath:              dbPath,
		GoogleAPIEndpoint:   googleServer.URL + "/",
		GoogleTaskListID:    "@default",
		TickTickAPIBaseURL:  ticktickServer.URL,
		TickTickAccessToken: "test-token",
		PollInterval:        time.Minute,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	app, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })

	if err := app.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Closes the database handle and returns nil when the app has a valid DB connection.
func TestAppClose(t *testing.T) {
	googleServer, ticktickServer := startMockServers(t)
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")

	cfg := config.Config{
		DBPath:              dbPath,
		GoogleAPIEndpoint:   googleServer.URL + "/",
		GoogleTaskListID:    "@default",
		TickTickAPIBaseURL:  ticktickServer.URL,
		TickTickAccessToken: "test-token",
		PollInterval:        time.Minute,
	}

	app, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	if err := app.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

// Creates HTTP test servers for the Google Tasks and TickTick APIs that respond to a one-task sync scenario (list, create, complete).
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
