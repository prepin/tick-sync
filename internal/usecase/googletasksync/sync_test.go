package googletasksync_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/prepin/tick-sync/internal/usecase/googletasksync"
	"github.com/prepin/tick-sync/internal/usecase/googletasksync/mocks"
)

// Keeps service expiry reminders in Google Tasks without copying, completing, or deleting them.
func TestHandleKeepsExpiryRemindersInGoogleTasks(t *testing.T) {
	t.Parallel()
	for _, action := range []googletasksync.PostSyncAction{
		googletasksync.PostSyncActionComplete, googletasksync.PostSyncActionDelete,
	} {
		t.Run(string(action), func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			google := mocks.NewMockGoogleTasksGateway(ctrl)
			ticktick := mocks.NewMockTickTickGateway(ctrl)
			repo := mocks.NewMockSyncedTaskRepository(ctrl)
			google.EXPECT().ListUncompleted(gomock.Any()).Return([]googletasksync.GoogleTaskView{
				{ID: "reminder-1", Notes: "Reconnect instructions\n" + googletasksync.TickTickExpiryReminderMarker},
			}, nil)
			result, err := googletasksync.New(google, ticktick, repo, action).Handle(t.Context())
			if err != nil {
				t.Fatalf("sync tasks: %v", err)
			}
			assertResult(t, result, googletasksync.SyncGoogleTasksToTickTickResult{Seen: 1, Skipped: 1})
		})
	}
}

// Does not exclude an ordinary task just because its notes mention the reminder marker inline.
func TestHandleImportsOrdinaryTaskMentioningReminderMarker(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	google := mocks.NewMockGoogleTasksGateway(ctrl)
	ticktick := mocks.NewMockTickTickGateway(ctrl)
	repo := mocks.NewMockSyncedTaskRepository(ctrl)
	google.EXPECT().ListUncompleted(gomock.Any()).Return([]googletasksync.GoogleTaskView{
		{ID: "ordinary-task", Notes: "Document " + googletasksync.TickTickExpiryReminderMarker + " behavior"},
	}, nil)
	repo.EXPECT().GetSyncState(gomock.Any(), "ordinary-task").Return(googletasksync.SyncState{}, false, nil)
	ticktick.EXPECT().
		CreateInboxTask(gomock.Any(), gomock.Any()).
		Return(googletasksync.TickTickTaskView{ID: "copy-1"}, nil)
	repo.EXPECT().SaveSyncedTask(gomock.Any(), gomock.Any()).Return(nil)
	google.EXPECT().Complete(gomock.Any(), "ordinary-task").Return(nil)
	repo.EXPECT().MarkGoogleTaskFinalized(gomock.Any(), "ordinary-task", gomock.Any()).Return(nil)
	result, err := googletasksync.New(google, ticktick, repo, googletasksync.PostSyncActionComplete).Handle(t.Context())
	if err != nil {
		t.Fatalf("sync task: %v", err)
	}
	assertResult(t, result, googletasksync.SyncGoogleTasksToTickTickResult{Seen: 1, Created: 1, Completed: 1})
}
