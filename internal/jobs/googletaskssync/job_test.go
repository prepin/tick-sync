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
func TestJobStartLogsAndReturnsOnExecuteFailure(t *testing.T) {
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

// Returns PostSyncActionComplete when the config value is empty or "complete".
func TestPostSyncActionFromConfigDefaultsToComplete(t *testing.T) {
	t.Parallel()
	got, err := PostSyncActionFromConfig("")
	if err != nil {
		t.Fatalf("post sync action from config: %v", err)
	}
	if got != googletasksync.PostSyncActionComplete {
		t.Fatalf("unexpected action: %s", got)
	}
}

// Returns PostSyncActionDelete when the config value is "delete".
func TestPostSyncActionFromConfigParsesDelete(t *testing.T) {
	t.Parallel()
	got, err := PostSyncActionFromConfig("delete")
	if err != nil {
		t.Fatalf("post sync action from config: %v", err)
	}
	if got != googletasksync.PostSyncActionDelete {
		t.Fatalf("unexpected action: %s", got)
	}
}

// Reports an error when the config value is not "complete" or "delete".
func TestPostSyncActionFromConfigReportsErrorForInvalidAction(t *testing.T) {
	t.Parallel()
	_, err := PostSyncActionFromConfig("archive")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "GOOGLE_POST_SYNC_ACTION") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Logs the sync summary at INFO level with all field values when there are no errors.
func TestPrintSyncSummaryLogsFieldValuesAtInfoLevel(t *testing.T) {
	buf := captureSlogOutput(t)

	PrintSyncSummary(googletasksync.SyncSummary{
		Seen:      4,
		Created:   3,
		Skipped:   1,
		Failed:    0,
		Completed: 3,
		Deleted:   0,
	})

	got := buf.String()
	if !strings.Contains(got, "level=INFO") {
		t.Fatalf("expected info level log, got %q", got)
	}
	if !strings.Contains(got, `seen=4`) {
		t.Fatalf("expected seen=4 in log, got %q", got)
	}
	if !strings.Contains(got, `created=3`) {
		t.Fatalf("expected created=3 in log, got %q", got)
	}
	if !strings.Contains(got, `skipped=1`) {
		t.Fatalf("expected skipped=1 in log, got %q", got)
	}
	if !strings.Contains(got, `failed=0`) {
		t.Fatalf("expected failed=0 in log, got %q", got)
	}
	if !strings.Contains(got, `completed=3`) {
		t.Fatalf("expected completed=3 in log, got %q", got)
	}
	if !strings.Contains(got, `deleted=0`) {
		t.Fatalf("expected deleted=0 in log, got %q", got)
	}
}

// Logs the sync summary at ERROR level with joined error messages when the sync encountered failures.
func TestPrintSyncSummaryLogsErrorsAtErrorLevel(t *testing.T) {
	buf := captureSlogOutput(t)

	PrintSyncSummary(googletasksync.SyncSummary{
		Seen:      1,
		Created:   0,
		Skipped:   0,
		Failed:    1,
		Completed: 0,
		Deleted:   0,
		Errors:    []error{errors.New("ticktick unavailable"), nil, errors.New("db write failed")},
	})

	got := buf.String()
	if !strings.Contains(got, "level=ERROR") {
		t.Fatalf("expected error level log, got %q", got)
	}
	if !strings.Contains(got, `errors="ticktick unavailable, db write failed"`) {
		t.Fatalf("expected errors in log, got %q", got)
	}
}

func captureSlogOutput(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return &buf
}
