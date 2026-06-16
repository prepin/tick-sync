package googletasksync_test

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
	"github.com/prepin/tick-sync/internal/usecase/googletasksync/mocks"
)

// Syncs a single unprocessed Google Task to TickTick, saves the record, and completes it on Google.
func TestUsecaseSyncGoogleTasksToTickTickCreatesRecordsAndCompletesTaskByDefault(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)
	syncedAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	task := googletasksync.GoogleTaskView{
		ID:      "google-1",
		Title:   "Buy milk",
		Notes:   "Remember lactose-free",
		Due:     "2026-06-12T00:00:00.000Z",
		Updated: "2026-06-10T10:00:00.000Z",
	}

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTaskView{task}, nil)
	repo.EXPECT().IsProcessed(ctx, "google-1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, googletasksync.CreateTickTickTaskInput{
		Title:              "Buy milk",
		Details:            "Remember lactose-free",
		Due:                "2026-06-12T00:00:00.000Z",
		SourceGoogleTaskID: "google-1",
	}).Return(googletasksync.TickTickTaskView{ID: "ticktick-1"}, nil)
	repo.EXPECT().SaveSyncedTask(ctx, googletasksync.SaveSyncedTaskParams{
		GoogleTaskID:   "google-1",
		GoogleUpdated:  "2026-06-10T10:00:00.000Z",
		GoogleTitle:    "Buy milk",
		TickTickTaskID: "ticktick-1",
		PostSyncAction: googletasksync.PostSyncActionComplete,
		SyncedAt:       syncedAt,
	}).Return(nil)
	google.EXPECT().Complete(ctx, "google-1").Return(nil)

	uc := googletasksync.New(
		google,
		ticktick,
		repo,
		googletasksync.PostSyncActionComplete,
		googletasksync.WithClock(func() time.Time { return syncedAt }),
	)

	result, err := uc.Handle(ctx)
	if err != nil {
		t.Fatalf("sync google tasks to ticktick: %v", err)
	}

	want := googletasksync.SyncGoogleTasksToTickTickResult{Seen: 1, Created: 1, Completed: 1}
	assertResult(t, result, want)
}

// Deletes the Google task instead of completing it when the post-sync action is set to "delete".
func TestUsecaseSyncGoogleTasksToTickTickDeletesTaskWhenConfigured(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)

	task := googletasksync.GoogleTaskView{ID: "google-1", Title: "Buy milk"}

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTaskView{task}, nil)
	repo.EXPECT().IsProcessed(ctx, "google-1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, gomock.Any()).Return(googletasksync.TickTickTaskView{ID: "ticktick-1"}, nil)
	repo.EXPECT().SaveSyncedTask(ctx, gomock.Any()).Return(nil)
	google.EXPECT().Delete(ctx, "google-1").Return(nil)

	result, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionDelete).Handle(ctx)
	if err != nil {
		t.Fatalf("sync google tasks to ticktick: %v", err)
	}

	want := googletasksync.SyncGoogleTasksToTickTickResult{Seen: 1, Created: 1, Deleted: 1}
	assertResult(t, result, want)
}

// Skips a task that was already synced in a previous run and does not create a TickTick task for it.
func TestUsecaseSyncGoogleTasksToTickTickSkipsAlreadyProcessedTask(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTaskView{{ID: "google-1"}}, nil)
	repo.EXPECT().IsProcessed(ctx, "google-1").Return(true, nil)

	result, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).Handle(ctx)
	if err != nil {
		t.Fatalf("sync google tasks to ticktick: %v", err)
	}

	want := googletasksync.SyncGoogleTasksToTickTickResult{Seen: 1, Skipped: 1}
	assertResult(t, result, want)
}

// Does not save a record or complete the task on Google when the TickTick API call fails.
func TestUsecaseSyncGoogleTasksToTickTickDoesNotCompleteTaskWhenTickTickCreationFails(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)
	createErr := errors.New("ticktick unavailable")

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTaskView{{ID: "google-1"}}, nil)
	repo.EXPECT().IsProcessed(ctx, "google-1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, gomock.Any()).Return(googletasksync.TickTickTaskView{}, createErr)

	result, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).Handle(ctx)
	if err == nil {
		t.Fatal("expected sync error")
	}

	want := googletasksync.SyncGoogleTasksToTickTickResult{Seen: 1, Failed: 1, Errors: []error{createErr}}
	assertResult(t, result, want)
}

// Does not complete the Google task when the synced record cannot be saved to the repo.
func TestUsecaseSyncGoogleTasksToTickTickDoesNotCompleteTaskWhenRepoRecordFails(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)
	repoErr := errors.New("db unavailable")

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTaskView{{ID: "google-1"}}, nil)
	repo.EXPECT().IsProcessed(ctx, "google-1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, gomock.Any()).Return(googletasksync.TickTickTaskView{ID: "ticktick-1"}, nil)
	repo.EXPECT().SaveSyncedTask(ctx, gomock.Any()).Return(repoErr)

	result, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).Handle(ctx)
	if err == nil {
		t.Fatal("expected sync error")
	}

	want := googletasksync.SyncGoogleTasksToTickTickResult{Seen: 1, Created: 1, Failed: 1, Errors: []error{repoErr}}
	assertResult(t, result, want)
}

// Continues syncing remaining tasks after a per-task error, reporting both successes and the failure in the result.
func TestUsecaseSyncGoogleTasksToTickTickContinuesAfterPerTaskError(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)
	createErr := errors.New("ticktick unavailable")

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTaskView{
		{ID: "google-1"},
		{ID: "google-2", Title: "Second task"},
	}, nil)

	repo.EXPECT().IsProcessed(ctx, "google-1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, gomock.Any()).Return(googletasksync.TickTickTaskView{}, createErr)

	repo.EXPECT().IsProcessed(ctx, "google-2").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, googletasksync.CreateTickTickTaskInput{
		Title:              "Second task",
		SourceGoogleTaskID: "google-2",
	}).Return(googletasksync.TickTickTaskView{ID: "ticktick-2"}, nil)
	repo.EXPECT().SaveSyncedTask(ctx, gomock.Any()).Return(nil)
	google.EXPECT().Complete(ctx, "google-2").Return(nil)

	result, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).Handle(ctx)
	if err == nil {
		t.Fatal("expected sync error")
	}

	want := googletasksync.SyncGoogleTasksToTickTickResult{
		Seen:      2,
		Created:   1,
		Failed:    1,
		Completed: 1,
		Errors:    []error{createErr},
	}
	assertResult(t, result, want)
}

// Returns an empty result and an error when the Google Tasks API itself is unavailable.
func TestUsecaseSyncGoogleTasksToTickTickReturnsListError(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)
	listErr := errors.New("google unavailable")

	google.EXPECT().ListUncompleted(ctx).Return(nil, listErr)

	result, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).Handle(ctx)
	if err == nil {
		t.Fatal("expected sync error")
	}

	assertResult(t, result, googletasksync.SyncGoogleTasksToTickTickResult{})
}

// assertResult checks that the result fields match expected values, ignoring the order of errors.
func assertResult(
	t *testing.T,
	got googletasksync.SyncGoogleTasksToTickTickResult,
	want googletasksync.SyncGoogleTasksToTickTickResult,
) {
	t.Helper()

	if got.Seen != want.Seen ||
		got.Created != want.Created ||
		got.Skipped != want.Skipped ||
		got.Failed != want.Failed ||
		got.Completed != want.Completed ||
		got.Deleted != want.Deleted ||
		len(got.Errors) != len(want.Errors) {
		t.Fatalf("unexpected result: got %+v, want %+v", got, want)
	}
}
