package googletasksync

import (
	"context"
	"time"
)

// SaveSyncedTaskParams is the usecase-owned input for recording a synced task.
type SaveSyncedTaskParams struct {
	GoogleTaskID   string
	GoogleUpdated  string
	GoogleTitle    string
	TickTickTaskID string
	PostSyncAction PostSyncAction
	SyncedAt       time.Time
}

// SyncState describes whether a Google task was copied to TickTick and finalized on Google.
type SyncState struct {
	GoogleTaskID      string
	TickTickTaskID    string
	PostSyncAction    PostSyncAction
	GoogleFinalizedAt time.Time
}

// IsGoogleFinalized returns true when the source Google task was completed or deleted.
func (s SyncState) IsGoogleFinalized() bool {
	return !s.GoogleFinalizedAt.IsZero()
}

// SyncedTaskRepository is the persistence port for idempotency records.
type SyncedTaskRepository interface {
	GetSyncState(ctx context.Context, googleTaskID string) (SyncState, bool, error)
	SaveSyncedTask(ctx context.Context, params SaveSyncedTaskParams) error
	MarkGoogleTaskFinalized(ctx context.Context, googleTaskID string, finalizedAt time.Time) error
}

//go:generate go tool mockgen -source=repository.go -destination=mocks/repository_mocks.go -package=mocks
