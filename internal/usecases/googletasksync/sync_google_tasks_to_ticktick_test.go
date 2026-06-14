package googletasksync_test

import (
	"errors"
	"testing"
	"time"

	googletasksync "github.com/prepin/tick-sync/internal/usecases/googletasksync"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync/mocks"
	"go.uber.org/mock/gomock"
)

// Syncs a single unprocessed Google Task to TickTick, saves the record, and completes it on Google.
func TestUsecaseSyncGoogleTasksToTickTickCreatesRecordsAndCompletesTaskByDefault(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	repo := mocks.NewMockSyncRepo(ctrl)
	syncedAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	task := googletasksync.GoogleTask{
		ID:      "google-1",
		Title:   "Buy milk",
		Notes:   "Remember lactose-free",
		Due:     "2026-06-12T00:00:00.000Z",
		Updated: "2026-06-10T10:00:00.000Z",
	}

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTask{task}, nil)
	repo.EXPECT().IsProcessed(ctx, "google-1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, googletasksync.TickTickTaskInput{
		Title:              "Buy milk",
		Details:            "Remember lactose-free",
		Due:                "2026-06-12T00:00:00.000Z",
		SourceGoogleTaskID: "google-1",
	}).Return(googletasksync.TickTickTask{ID: "ticktick-1"}, nil)
	repo.EXPECT().SaveSyncedTask(ctx, googletasksync.SyncedTaskRecord{
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

	summary, err := uc.SyncGoogleTasksToTickTick(ctx)
	if err != nil {
		t.Fatalf("sync google tasks to ticktick: %v", err)
	}

	want := googletasksync.SyncSummary{Seen: 1, Created: 1, Completed: 1}
	assertSummary(t, summary, want)
}

// Deletes the Google task instead of completing it when the post-sync action is set to "delete".
func TestUsecaseSyncGoogleTasksToTickTickDeletesTaskWhenConfigured(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	repo := mocks.NewMockSyncRepo(ctrl)

	task := googletasksync.GoogleTask{ID: "google-1", Title: "Buy milk"}

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTask{task}, nil)
	repo.EXPECT().IsProcessed(ctx, "google-1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, gomock.Any()).Return(googletasksync.TickTickTask{ID: "ticktick-1"}, nil)
	repo.EXPECT().SaveSyncedTask(ctx, gomock.Any()).Return(nil)
	google.EXPECT().Delete(ctx, "google-1").Return(nil)

	summary, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionDelete).SyncGoogleTasksToTickTick(ctx)
	if err != nil {
		t.Fatalf("sync google tasks to ticktick: %v", err)
	}

	want := googletasksync.SyncSummary{Seen: 1, Created: 1, Deleted: 1}
	assertSummary(t, summary, want)
}

// Skips a task that was already synced in a previous run and does not create a TickTick task for it.
func TestUsecaseSyncGoogleTasksToTickTickSkipsAlreadyProcessedTask(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	repo := mocks.NewMockSyncRepo(ctrl)

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTask{{ID: "google-1"}}, nil)
	repo.EXPECT().IsProcessed(ctx, "google-1").Return(true, nil)

	summary, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).SyncGoogleTasksToTickTick(ctx)
	if err != nil {
		t.Fatalf("sync google tasks to ticktick: %v", err)
	}

	want := googletasksync.SyncSummary{Seen: 1, Skipped: 1}
	assertSummary(t, summary, want)
}

// Does not save a record or complete the task on Google when the TickTick API call fails.
func TestUsecaseSyncGoogleTasksToTickTickDoesNotCompleteTaskWhenTickTickCreationFails(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	repo := mocks.NewMockSyncRepo(ctrl)
	createErr := errors.New("ticktick unavailable")

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTask{{ID: "google-1"}}, nil)
	repo.EXPECT().IsProcessed(ctx, "google-1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, gomock.Any()).Return(googletasksync.TickTickTask{}, createErr)

	summary, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).SyncGoogleTasksToTickTick(ctx)
	if err == nil {
		t.Fatal("expected sync error")
	}

	want := googletasksync.SyncSummary{Seen: 1, Failed: 1, Errors: []error{createErr}}
	assertSummary(t, summary, want)
}

// Does not complete the Google task when the synced record cannot be saved to the repo.
func TestUsecaseSyncGoogleTasksToTickTickDoesNotCompleteTaskWhenRepoRecordFails(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	repo := mocks.NewMockSyncRepo(ctrl)
	repoErr := errors.New("db unavailable")

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTask{{ID: "google-1"}}, nil)
	repo.EXPECT().IsProcessed(ctx, "google-1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, gomock.Any()).Return(googletasksync.TickTickTask{ID: "ticktick-1"}, nil)
	repo.EXPECT().SaveSyncedTask(ctx, gomock.Any()).Return(repoErr)

	summary, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).SyncGoogleTasksToTickTick(ctx)
	if err == nil {
		t.Fatal("expected sync error")
	}

	want := googletasksync.SyncSummary{Seen: 1, Created: 1, Failed: 1, Errors: []error{repoErr}}
	assertSummary(t, summary, want)
}

// Continues syncing remaining tasks after a per-task error, reporting both successes and the failure in the summary.
func TestUsecaseSyncGoogleTasksToTickTickContinuesAfterPerTaskError(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	repo := mocks.NewMockSyncRepo(ctrl)
	createErr := errors.New("ticktick unavailable")

	google.EXPECT().ListUncompleted(ctx).Return([]googletasksync.GoogleTask{
		{ID: "google-1"},
		{ID: "google-2", Title: "Second task"},
	}, nil)

	repo.EXPECT().IsProcessed(ctx, "google-1").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, gomock.Any()).Return(googletasksync.TickTickTask{}, createErr)

	repo.EXPECT().IsProcessed(ctx, "google-2").Return(false, nil)
	ticktick.EXPECT().CreateInboxTask(ctx, googletasksync.TickTickTaskInput{
		Title:              "Second task",
		SourceGoogleTaskID: "google-2",
	}).Return(googletasksync.TickTickTask{ID: "ticktick-2"}, nil)
	repo.EXPECT().SaveSyncedTask(ctx, gomock.Any()).Return(nil)
	google.EXPECT().Complete(ctx, "google-2").Return(nil)

	summary, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).SyncGoogleTasksToTickTick(ctx)
	if err == nil {
		t.Fatal("expected sync error")
	}

	want := googletasksync.SyncSummary{Seen: 2, Created: 1, Failed: 1, Completed: 1, Errors: []error{createErr}}
	assertSummary(t, summary, want)
}

// Returns an empty summary and an error when the Google Tasks API itself is unavailable.
func TestUsecaseSyncGoogleTasksToTickTickReturnsListError(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)

	google := mocks.NewMockGoogleTasksClient(ctrl)
	ticktick := mocks.NewMockTickTickClient(ctrl)
	repo := mocks.NewMockSyncRepo(ctrl)
	listErr := errors.New("google unavailable")

	google.EXPECT().ListUncompleted(ctx).Return(nil, listErr)

	summary, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).SyncGoogleTasksToTickTick(ctx)
	if err == nil {
		t.Fatal("expected sync error")
	}

	assertSummary(t, summary, googletasksync.SyncSummary{})
}

// assertSummary checks that the SyncSummary fields match expected values, ignoring the order of errors.
func assertSummary(t *testing.T, got googletasksync.SyncSummary, want googletasksync.SyncSummary) {
	t.Helper()

	if got.Seen != want.Seen ||
		got.Created != want.Created ||
		got.Skipped != want.Skipped ||
		got.Failed != want.Failed ||
		got.Completed != want.Completed ||
		got.Deleted != want.Deleted ||
		len(got.Errors) != len(want.Errors) {
		t.Fatalf("unexpected summary: got %+v, want %+v", got, want)
	}
}
