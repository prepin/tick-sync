package googletasksync

import "context"

// CreateTickTickTaskInput is the usecase-owned input for creating a TickTick task.
type CreateTickTickTaskInput struct {
	Title              string
	Details            string
	Due                string
	SourceGoogleTaskID string
}

// GoogleTasksGateway is the outbound port for reading and mutating Google tasks.
type GoogleTasksGateway interface {
	ListUncompleted(ctx context.Context) ([]GoogleTaskView, error)
	Complete(ctx context.Context, taskID string) error
	Delete(ctx context.Context, taskID string) error
}

// TickTickGateway is the outbound port for creating TickTick tasks.
type TickTickGateway interface {
	CreateInboxTask(ctx context.Context, input CreateTickTickTaskInput) (TickTickTaskView, error)
}

//go:generate go tool mockgen -source=gateway.go -destination=mocks/gateway_mocks.go -package=mocks
