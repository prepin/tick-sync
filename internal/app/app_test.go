package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	_ "modernc.org/sqlite"

	"github.com/prepin/tick-sync/internal/config"
	googletasksyncjob "github.com/prepin/tick-sync/internal/entrypoints/cron/googletasksync"
	tickticktokenreminderjob "github.com/prepin/tick-sync/internal/entrypoints/cron/tickticktokenreminder"
	googletasksrepo "github.com/prepin/tick-sync/internal/infra/sqlite/googletasks"
	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
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

// Persists expiry reminders across app restarts and allows another reminder for a replacement TickTick token.
func TestAppExpiryReminderSurvivesRestartAndResetsOnReconnect(t *testing.T) {
	t.Parallel()
	var googleCreations atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tasks/v1/lists/@default/tasks" {
			t.Errorf("unexpected provider request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		googleCreations.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"google-reminder"}`))
	}))
	t.Cleanup(server.Close)
	cfg := config.Config{
		DBPath:             filepath.Join(t.TempDir(), "tick-sync.db"),
		GoogleAPIEndpoint:  server.URL + "/",
		GoogleTaskListID:   "@default",
		TickTickAPIBaseURL: server.URL + "/ticktick",
		Location:           time.UTC,
	}
	first, err := New(t.Context(), cfg)
	if err != nil {
		t.Fatalf("create first app: %v", err)
	}
	t.Cleanup(func() { first.Close() })
	tokens, err := oauthtokens.New(first.db)
	if err != nil {
		t.Fatalf("open token repository: %v", err)
	}
	token := oauthtokens.Token{
		AccessToken: "expired-token", ExpiresAt: time.Now().Add(-time.Hour),
		RefreshReminderTaskID: "existing-ticktick-reminder",
	}
	if err := tokens.Save(t.Context(), oauthtokens.ProviderTickTick, token); err != nil {
		t.Fatalf("save expired token: %v", err)
	}
	if err := first.jobs[1].(*tickticktokenreminderjob.Job).Execute(t.Context()); err != nil {
		t.Fatalf("create first reminder: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first app: %v", err)
	}
	second, err := New(t.Context(), cfg)
	if err != nil {
		t.Fatalf("restart app: %v", err)
	}
	t.Cleanup(func() { second.Close() })
	job := second.jobs[1].(*tickticktokenreminderjob.Job)
	if err := job.Execute(t.Context()); err != nil {
		t.Fatalf("check reminder after restart: %v", err)
	}
	if got := googleCreations.Load(); got != 1 {
		t.Fatalf("expected one reminder after restart, got %d", got)
	}
	tokens, err = oauthtokens.New(second.db)
	if err != nil {
		t.Fatalf("reopen token repository: %v", err)
	}
	token.AccessToken = "replacement-token"
	if err := tokens.Save(t.Context(), oauthtokens.ProviderTickTick, token); err != nil {
		t.Fatalf("save replacement token: %v", err)
	}
	if err := job.Execute(t.Context()); err != nil {
		t.Fatalf("create replacement token reminder: %v", err)
	}
	if got := googleCreations.Load(); got != 2 {
		t.Fatalf("expected a reminder for each token, got %d", got)
	}
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
