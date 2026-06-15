package app

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prepin/tick-sync/internal/config"
	googletasksrepo "github.com/prepin/tick-sync/internal/infra/sqlite/googletasks"
	googletaskssyncjob "github.com/prepin/tick-sync/internal/jobs/googletaskssync"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync/mocks"
	"go.uber.org/mock/gomock"
	_ "modernc.org/sqlite"
)

// Does not create an app when the database path is a directory or unwritable.
func TestNewRejectsDBOpenFailure(t *testing.T) {
	ctx := t.Context()
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
	ctx := t.Context()
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
	ctrl := gomock.NewController(t)
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo, err := googletasksrepo.NewGoogleTasksRepo(t.Context(), db)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, nil)

	uc := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete)
	job := googletaskssyncjob.New(uc, time.Minute)

	cfg := config.Config{DBPath: dbPath, PollInterval: time.Minute}
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	application, err := New(ctx, cfg, WithJobs([]Runner{job}))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { application.Close() })

	if err := application.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Closes the database handle and returns nil when the app has a valid DB connection.
func TestAppClose(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")
	cfg := config.Config{DBPath: dbPath}

	application, err := New(t.Context(), cfg, WithJobs([]Runner{}))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	if err := application.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
