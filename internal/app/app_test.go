package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/testutil"
)

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
	if !strings.Contains(err.Error(), "create google tasks repo") {
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

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	app, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { app.Close() })

	if err := app.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Closes the database handle and returns nil when the app has a valid DB connection.
func TestAppClose(t *testing.T) {
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

	app, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	if err := app.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
