package googletasksync

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Handle runs one sync cycle.
func (u *SyncGoogleTasksToTickTickUseCase) Handle(ctx context.Context) (SyncGoogleTasksToTickTickResult, error) {
	googleTasks, err := u.google.ListUncompleted(ctx)
	if err != nil {
		return SyncGoogleTasksToTickTickResult{}, fmt.Errorf("list uncompleted google tasks: %w", err)
	}

	result := SyncGoogleTasksToTickTickResult{Seen: len(googleTasks)}
	for _, googleTask := range googleTasks {
		if err := u.syncTaskToTickTick(ctx, googleTask, &result); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err)
		}
	}

	return result, errors.Join(result.Errors...)
}

func (u *SyncGoogleTasksToTickTickUseCase) syncTaskToTickTick(
	ctx context.Context,
	googleTask GoogleTaskView,
	result *SyncGoogleTasksToTickTickResult,
) error {
	if u.shouldDelayTodayImport(googleTask) {
		result.Delayed++
		return nil
	}

	state, found, err := u.repo.GetSyncState(ctx, googleTask.ID)
	if err != nil {
		return fmt.Errorf("get sync state for google task %s: %w", googleTask.ID, err)
	}
	if found && state.IsGoogleFinalized() {
		result.Skipped++
		return nil
	}
	if found {
		return u.finalizeGoogleTask(ctx, googleTask.ID, state.PostSyncAction, result)
	}

	tickTickTask, err := u.ticktick.CreateInboxTask(ctx, CreateTickTickTaskInput{
		Title:              googleTask.Title,
		Details:            googleTask.Notes,
		Due:                googleTask.Due,
		SourceGoogleTaskID: googleTask.ID,
	})
	if err != nil {
		return fmt.Errorf("create ticktick task for google task %s: %w", googleTask.ID, err)
	}
	result.Created++

	if err := u.repo.SaveSyncedTask(ctx, SaveSyncedTaskParams{
		GoogleTaskID:   googleTask.ID,
		GoogleUpdated:  googleTask.Updated,
		GoogleTitle:    googleTask.Title,
		TickTickTaskID: tickTickTask.ID,
		PostSyncAction: u.postSyncAction,
		SyncedAt:       u.now(),
	}); err != nil {
		return fmt.Errorf("record processed google task %s: %w", googleTask.ID, err)
	}

	return u.finalizeGoogleTask(ctx, googleTask.ID, u.postSyncAction, result)
}

func (u *SyncGoogleTasksToTickTickUseCase) finalizeGoogleTask(
	ctx context.Context,
	googleTaskID string,
	postSyncAction PostSyncAction,
	result *SyncGoogleTasksToTickTickResult,
) error {
	switch postSyncAction {
	case PostSyncActionDelete:
		if err := u.google.Delete(ctx, googleTaskID); err != nil {
			return fmt.Errorf("delete google task %s: %w", googleTaskID, err)
		}
		result.Deleted++
	case PostSyncActionComplete:
		if err := u.google.Complete(ctx, googleTaskID); err != nil {
			return fmt.Errorf("complete google task %s: %w", googleTaskID, err)
		}
		result.Completed++
	default:
		return fmt.Errorf("unsupported post sync action %q", postSyncAction)
	}

	if err := u.repo.MarkGoogleTaskFinalized(ctx, googleTaskID, u.now()); err != nil {
		return fmt.Errorf("record finalized google task %s: %w", googleTaskID, err)
	}

	return nil
}

func (u *SyncGoogleTasksToTickTickUseCase) shouldDelayTodayImport(googleTask GoogleTaskView) bool {
	if !u.delayTodayImports || googleTask.Due == "" {
		return false
	}

	due, err := time.Parse(time.RFC3339Nano, googleTask.Due)
	if err != nil {
		return false
	}

	dueYear, dueMonth, dueDay := due.Date()
	nowYear, nowMonth, nowDay := u.now().In(u.location).Date()

	return dueYear == nowYear && dueMonth == nowMonth && dueDay == nowDay
}
