package googletasksync

import (
	"context"
	"errors"
	"fmt"

	"github.com/prepin/tick-sync/internal/consts"
)

type PostSyncAction = consts.PostSyncAction

const (
	PostSyncActionComplete = consts.PostSyncActionComplete
	PostSyncActionDelete   = consts.PostSyncActionDelete
)

func (u *Usecase) SyncGoogleTasksToTickTick(ctx context.Context) (SyncSummary, error) {
	googleTasks, err := u.google.ListUncompleted(ctx)
	if err != nil {
		return SyncSummary{}, fmt.Errorf("list uncompleted google tasks: %w", err)
	}

	summary := SyncSummary{Seen: len(googleTasks)}
	for _, googleTask := range googleTasks {
		if err := u.syncTaskToTickTick(ctx, googleTask, &summary); err != nil {
			summary.Failed++
			summary.Errors = append(summary.Errors, err)
		}
	}

	return summary, errors.Join(summary.Errors...)
}

func (u *Usecase) syncTaskToTickTick(ctx context.Context, googleTask GoogleTask, summary *SyncSummary) error {
	processed, err := u.repo.IsProcessed(ctx, googleTask.ID)
	if err != nil {
		return fmt.Errorf("check processed google task %s: %w", googleTask.ID, err)
	}
	if processed {
		summary.Skipped++
		return nil
	}

	tickTickTask, err := u.ticktick.CreateInboxTask(ctx, TickTickTaskInput{
		Title:              googleTask.Title,
		Details:            googleTask.Notes,
		Due:                googleTask.Due,
		SourceGoogleTaskID: googleTask.ID,
	})
	if err != nil {
		return fmt.Errorf("create ticktick task for google task %s: %w", googleTask.ID, err)
	}
	summary.Created++

	if err := u.repo.SaveSyncedTask(ctx, SyncedTaskRecord{
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
		summary.Deleted++
	default:
		if err := u.google.Complete(ctx, googleTask.ID); err != nil {
			return fmt.Errorf("complete google task %s: %w", googleTask.ID, err)
		}
		summary.Completed++
	}

	return nil
}
