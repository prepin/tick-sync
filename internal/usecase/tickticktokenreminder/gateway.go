package tickticktokenreminder

import (
	"context"

	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

// TickTickGateway is the outbound port for creating the refresh reminder task.
type TickTickGateway interface {
	CreateInboxTask(ctx context.Context, input googletasksync.CreateTickTickTaskInput) (googletasksync.TickTickTaskView, error)
}

//go:generate go tool mockgen -source=gateway.go -destination=mocks/gateway_mocks.go -package=mocks
