package googletaskssyncjob

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	googletasksync "github.com/prepin/tick-sync/internal/usecases/googletasksync"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync/mocks"
	"go.uber.org/mock/gomock"
)

// Returns "google-tasks-sync" as the job name when queried.
func TestJobName(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	job := New(googletasksync.New(mocks.NewMockGoogleTasksClient(ctrl), mocks.NewMockTickTickClient(ctrl), mocks.NewMockSyncStore(ctrl), ""), time.Minute)
	if got := job.Name(); got != "google-tasks-sync" {
		t.Fatalf("unexpected name: %s", got)
	}
}

// Runs the initial sync, then stops on context cancellation without executing another tick.
func TestJobStartExecutesSyncAndStopsOnCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	store := mocks.NewMockSyncStore(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, nil)
	store.EXPECT().IsProcessed(gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)

	uc := googletasksync.New(google, ticktick, store, googletasksync.PostSyncActionComplete)
	job := New(uc, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	job.Start(ctx)
	cancel()
	time.Sleep(50 * time.Millisecond)
}

// Does not enter the ticker loop when the initial Execute call fails: the goroutine exits after the single failed attempt.
func TestJobStartDoesNotEnterTickerLoopOnExecuteFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	store := mocks.NewMockSyncStore(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, errors.New("google unavailable"))

	uc := googletasksync.New(google, ticktick, store, googletasksync.PostSyncActionComplete)
	job := New(uc, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	job.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
}

// Returns the SyncSummary from the usecase and reports a nil error when all tasks sync successfully.
func TestJobExecuteReportsSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	store := mocks.NewMockSyncStore(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, nil)
	store.EXPECT().IsProcessed(gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)

	uc := googletasksync.New(google, ticktick, store, googletasksync.PostSyncActionComplete)
	job := New(uc, time.Minute)

	if err := job.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Logs the sync summary at INFO level with all field values when there are no errors, including the job name.
func TestJobExecuteLogsSummaryWithJobName(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	store := mocks.NewMockSyncStore(ctrl)

	google.EXPECT().ListUncompleted(gomock.Any()).Return(nil, nil)
	store.EXPECT().IsProcessed(gomock.Any(), gomock.Any()).AnyTimes().Return(false, nil)

	uc := googletasksync.New(google, ticktick, store, googletasksync.PostSyncActionComplete)
	job := New(uc, time.Minute)

	buf := captureSlogOutput(t)

	if err := job.Execute(context.Background()); err != nil {
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

// Logs the sync summary at ERROR level with joined error messages when the sync encountered failures.
func TestJobExecuteLogsErrorsAtErrorLevel(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	store := mocks.NewMockSyncStore(ctrl)
	createErr := errors.New("ticktick unavailable")

	google.EXPECT().ListUncompleted(gomock.Any()).Return([]googletasksync.GoogleTask{{ID: "g1"}}, nil)
	store.EXPECT().IsProcessed(gomock.Any(), "g1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(gomock.Any(), gomock.Any()).Return(googletasksync.TickTickTask{}, createErr)

	uc := googletasksync.New(google, ticktick, store, googletasksync.PostSyncActionComplete)
	job := New(uc, time.Minute)

	buf := captureSlogOutput(t)

	_ = job.Execute(context.Background())

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
