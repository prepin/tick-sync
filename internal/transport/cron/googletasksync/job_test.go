package googletasksync_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	googletasksync "github.com/prepin/tick-sync/internal/application/googletasksync"
	"github.com/prepin/tick-sync/internal/application/googletasksync/mocks"
	cron "github.com/prepin/tick-sync/internal/transport/cron/googletasksync"
	"go.uber.org/mock/gomock"
)

// Returns "google-tasks-sync" as the job name when queried.
func TestJobName(t *testing.T) {
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

// Does not enter the ticker loop when the initial Execute call fails: the goroutine exits after the single failed attempt.
func TestJobStartDoesNotEnterTickerLoopOnExecuteFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, errors.New("google unavailable"))

	uc := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete)
	job := cron.New(uc, time.Minute)

	ctx, cancel := context.WithCancel(t.Context())
	job.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
}

// Returns the result from the usecase and reports a nil error when all tasks sync successfully.
func TestJobExecuteReportsSuccess(t *testing.T) {
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
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, nil)
	repo.EXPECT().IsProcessed(gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)

	uc := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete)
	job := cron.New(uc, time.Minute)

	buf := captureSlogOutput(t)

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
	job := cron.New(uc, time.Minute)

	buf := captureSlogOutput(t)

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

// Captures slog output into a byte buffer for assertion in tests, restoring the previous default handler after the test.
func captureSlogOutput(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return &buf
}
