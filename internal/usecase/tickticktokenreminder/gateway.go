package tickticktokenreminder

import (
	"context"
	"time"

	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

// CreateGoogleTaskInput describes a token reminder in Google Tasks.
type CreateGoogleTaskInput struct {
	Title string
	Notes string
	Due   time.Time
}

// GoogleTasksGateway creates the reminder when TickTick is no longer usable.
type GoogleTasksGateway interface {
	CreateTask(ctx context.Context, input CreateGoogleTaskInput) (string, error)
}

// TickTickGateway is the outbound port for creating the refresh reminder task.
type TickTickGateway interface {
	CreateInboxTask(
		ctx context.Context,
		input googletasksync.CreateTickTickTaskInput,
	) (googletasksync.TickTickTaskView, error)
}

//go:generate go tool mockgen -source=gateway.go -destination=mocks/gateway_mocks.go -package=mocks
