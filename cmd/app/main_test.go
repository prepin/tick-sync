package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/prepin/tick-sync/internal/app"
	"github.com/prepin/tick-sync/internal/config"
)

// Runs the app with mock servers, executes one sync, and returns nil when the context is cancelled.
func TestMainRunsSyncAndStopsOnContextCancel(t *testing.T) {
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

	application, err := app.New(ctx, cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { _ = application.Close() })

	if err := application.Run(ctx); err != nil {
		t.Fatalf("app run: %v", err)
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
