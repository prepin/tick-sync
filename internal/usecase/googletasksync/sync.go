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

	processed, err := u.repo.IsProcessed(ctx, googleTask.ID)
	if err != nil {
		return fmt.Errorf("check processed google task %s: %w", googleTask.ID, err)
	}
	if processed {
		result.Skipped++
		return nil
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

	switch u.postSyncAction {
	case PostSyncActionDelete:
		if err := u.google.Delete(ctx, googleTask.ID); err != nil {
			return fmt.Errorf("delete google task %s: %w", googleTask.ID, err)
		}
		result.Deleted++
	case PostSyncActionComplete:
		if err := u.google.Complete(ctx, googleTask.ID); err != nil {
			return fmt.Errorf("complete google task %s: %w", googleTask.ID, err)
		}
		result.Completed++
	default:
		return fmt.Errorf("unsupported post sync action %q", u.postSyncAction)
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
