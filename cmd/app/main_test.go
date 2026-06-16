package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/prepin/tick-sync/internal/app"
	"github.com/prepin/tick-sync/internal/config"
	googletasksyncjob "github.com/prepin/tick-sync/internal/entrypoints/cron/googletasksync"
	googletasksrepo "github.com/prepin/tick-sync/internal/infra/sqlite/googletasks"
	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
	"github.com/prepin/tick-sync/internal/usecase/googletasksync/mocks"
	"go.uber.org/mock/gomock"
	_ "modernc.org/sqlite"
)

// Runs the app with mock clients, executes one sync, and returns nil when the context is cancelled.
func TestMainRunsSyncAndStopsOnContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	dbPath := filepath.Join(t.TempDir(), "tick-sync.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo, err := googletasksrepo.New(t.Context(), db)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, nil)

	uc := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete)
	job := googletasksyncjob.New(uc, time.Minute)

	cfg := config.Config{DBPath: dbPath, PollInterval: time.Minute}
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	application, err := app.New(ctx, cfg, app.WithJobs([]app.JobsRunner{job}))
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { _ = application.Close() })

	if err := application.Run(ctx); err != nil {
		t.Fatalf("app run: %v", err)
	}
}
