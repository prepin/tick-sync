package googletasksync

import "context"

//go:generate go run go.uber.org/mock/mockgen -source=ports.go -destination=mocks/mocks.go -package=mocks

type GoogleTasksClient interface {
	ListUncompleted(ctx context.Context) ([]GoogleTask, error)
	Complete(ctx context.Context, taskID string) error
	Delete(ctx context.Context, taskID string) error
}

type TickTickClient interface {
	CreateInboxTask(ctx context.Context, task TickTickTaskInput) (TickTickTask, error)
}

type SyncStore interface {
	IsProcessed(ctx context.Context, googleTaskID string) (bool, error)
	MarkProcessed(ctx context.Context, record SyncedTaskRecord) error
}
