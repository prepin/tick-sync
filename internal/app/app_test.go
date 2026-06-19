package app

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	_ "modernc.org/sqlite"

	"github.com/prepin/tick-sync/internal/config"
	googletasksyncjob "github.com/prepin/tick-sync/internal/entrypoints/cron/googletasksync"
	googletasksrepo "github.com/prepin/tick-sync/internal/infra/sqlite/googletasks"
	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
	"github.com/prepin/tick-sync/internal/usecase/googletasksync/mocks"
)

// Does not create an app when the database path is a directory or unwritable.
func TestNewRejectsDBOpenFailure(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	dir := t.TempDir()
	cfg := config.Config{
		DBPath: dir,
	}

	_, err := New(ctx, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "run sqlite migrations") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Creates the app before TickTick is connected so the local auth page can store the token later.
func TestNewAllowsMissingTickTickAccessToken(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")
	cfg := config.Config{
		DBPath:            dbPath,
		GoogleAPIEndpoint: "https://example.com/",
		PollInterval:      time.Minute,
	}

	application, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { application.Close() })
}

// Runs the sync job once and returns nil when the context is cancelled after the first execution.
func TestAppRunStopsOnContextCancel(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo, err := googletasksrepo.New(db)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, nil)

	uc := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete)
	job := googletasksyncjob.New(uc, time.Minute)

	cfg := config.Config{DBPath: dbPath, HTTPAddr: "127.0.0.1:0", PollInterval: time.Minute}
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

// Returns an error and cancels sibling runners when one supervised runner fails.
func TestAppRunReturnsRunnerFailureAndCancelsSiblings(t *testing.T) {
	t.Parallel()
	runnerErr := errors.New("runner failed")
	siblingCancelled := make(chan struct{})
	application := newTestAppWithRunners(t, []Runner{
		testRunner{run: func(context.Context) error { return runnerErr }},
		testRunner{run: func(ctx context.Context) error {
			<-ctx.Done()
			close(siblingCancelled)
			return nil
		}},
	})

	err := application.Run(t.Context())
	if err == nil {
		t.Fatal("expected runner error")
	}
	if !errors.Is(err, runnerErr) {
		t.Fatalf("expected runner failure, got %v", err)
	}
	select {
	case <-siblingCancelled:
	case <-time.After(time.Second):
		t.Fatal("expected sibling runner cancellation")
	}
}

// Returns an error when a supervised runner stops before shutdown without reporting a failure.
func TestAppRunReportsUnexpectedRunnerStop(t *testing.T) {
	t.Parallel()
	application := newTestAppWithRunners(t, []Runner{
		testRunner{run: func(context.Context) error { return nil }},
	})

	err := application.Run(t.Context())
	if err == nil {
		t.Fatal("expected unexpected stop error")
	}
	if !strings.Contains(err.Error(), "stopped unexpectedly") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Closes the database handle and returns nil when the app has a valid DB connection.
func TestAppClose(t *testing.T) {
	t.Parallel()
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

// newTestAppWithRunners creates an app with only the provided supervised runners and no HTTP server.
func newTestAppWithRunners(t *testing.T, runners []Runner) *App {
	t.Helper()
	application, err := New(
		t.Context(),
		config.Config{DBPath: filepath.Join(t.TempDir(), "tick-sync.db")},
		WithJobs(runners),
	)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	application.web = nil
	t.Cleanup(func() { application.Close() })
	return application
}

type testRunner struct {
	run func(context.Context) error
}

func (r testRunner) Run(ctx context.Context) error {
	return r.run(ctx)
}
