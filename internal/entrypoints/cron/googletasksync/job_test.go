package googletasksync_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	cron "github.com/prepin/tick-sync/internal/entrypoints/cron/googletasksync"
	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
	"github.com/prepin/tick-sync/internal/usecase/googletasksync/mocks"
)

// Returns "google-tasks-sync" as the job name when queried.
func TestJobName(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	job := cron.New(googletasksync.New(
		mocks.NewMockGoogleTasksGateway(ctrl),
		mocks.NewMockTickTickGateway(ctrl),
		mocks.NewMockSyncedTaskRepository(ctrl),
		googletasksync.PostSyncActionComplete,
	), time.Minute)
	if got := job.Name(); got != "google-tasks-sync" {
		t.Fatalf("unexpected name: %s", got)
	}
}

// Runs the initial sync, then stops on context cancellation without executing another tick.
func TestJobStartExecutesSyncAndStopsOnCancel(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, nil)
	repo.EXPECT().IsProcessed(gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)

	uc := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete)
	job := cron.New(uc, time.Minute)

	ctx, cancel := context.WithCancel(t.Context())
	job.Start(ctx)
	cancel()
	time.Sleep(50 * time.Millisecond)
}

// Continues polling after the initial sync fails so a later auth token can unblock future ticks.
func TestJobStartContinuesAfterInitialExecuteFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	gomock.InOrder(
		google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, errors.New("google unavailable")),
		google.EXPECT().ListUncompleted(gomock.Any()).DoAndReturn(func(context.Context) ([]googletasksync.GoogleTaskView, error) {
			cancel()
			close(done)
			return nil, nil
		}),
	)
	repo.EXPECT().IsProcessed(gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)

	uc := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete)
	job := cron.New(uc, 10*time.Millisecond)

	job.Start(ctx)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected job to continue after initial failure")
	}
}

// Returns the result from the usecase and reports a nil error when all tasks sync successfully.
func TestJobExecuteReportsSuccess(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, nil)
	repo.EXPECT().IsProcessed(gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)

	uc := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete)
	job := cron.New(uc, time.Minute)

	if err := job.Execute(t.Context()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Logs the sync result at INFO level with all field values when there are no errors, including the job name.
func TestJobExecuteLogsResultWithJobName(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, nil)
	repo.EXPECT().IsProcessed(gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)

	uc := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete)
	buf, logger := newTestLogger(t)
	job := cron.New(uc, time.Minute, cron.WithLogger(logger))

	if err := job.Execute(t.Context()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `job=google-tasks-sync`) {
		t.Fatalf("expected job name in log, got %q", got)
	}
	if !strings.Contains(got, "level=INFO") {
		t.Fatalf("expected info level log, got %q", got)
	}
	if !strings.Contains(got, `seen=0`) {
		t.Fatalf("expected seen=0 in log, got %q", got)
	}
}

// Logs the sync result at ERROR level with joined error messages when the sync encountered failures.
func TestJobExecuteLogsErrorsAtErrorLevel(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)
	createErr := errors.New("ticktick unavailable")

	google.EXPECT().ListUncompleted(gomock.Any()).Return([]googletasksync.GoogleTaskView{{ID: "g1"}}, nil)
	repo.EXPECT().IsProcessed(gomock.Any(), "g1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(gomock.Any(), gomock.Any()).Return(googletasksync.TickTickTaskView{}, createErr)

	uc := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete)
	buf, logger := newTestLogger(t)
	job := cron.New(uc, time.Minute, cron.WithLogger(logger))

	_ = job.Execute(t.Context())

	got := buf.String()
	if !strings.Contains(got, `job=google-tasks-sync`) {
		t.Fatalf("expected job name in log, got %q", got)
	}
	if !strings.Contains(got, "level=ERROR") {
		t.Fatalf("expected error level log, got %q", got)
	}
	if !strings.Contains(got, `ticktick unavailable`) {
		t.Fatalf("expected errors in log, got %q", got)
	}
}

// newTestLogger returns a logger writing to a byte buffer for assertion in tests.
func newTestLogger(t *testing.T) (*bytes.Buffer, *slog.Logger) {
	t.Helper()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	return &buf, logger
}
