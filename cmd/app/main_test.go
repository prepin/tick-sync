package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/prepin/tick-sync/internal/app"
	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/testutil"
)

// Runs the app with mock servers, executes one sync, and returns nil when the context is cancelled.
func TestMainRunsSyncAndStopsOnContextCancel(t *testing.T) {
	googleServer, ticktickServer := testutil.StartMockServers(t)
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")

	cfg := config.Config{
		DBPath:              dbPath,
		GoogleAPIEndpoint:   googleServer.URL + "/",
		GoogleTaskListID:    "@default",
		TickTickAPIBaseURL:  ticktickServer.URL,
		TickTickAccessToken: "test-token",
		PollInterval:        time.Minute,
	}

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
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
