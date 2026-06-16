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

// SyncedTaskRepository is the persistence port for idempotency records.
type SyncedTaskRepository interface {
	IsProcessed(ctx context.Context, googleTaskID string) (bool, error)
	SaveSyncedTask(ctx context.Context, params SaveSyncedTaskParams) error
}

//go:generate go tool mockgen -source=repository.go -destination=mocks/repository_mocks.go -package=mocks
