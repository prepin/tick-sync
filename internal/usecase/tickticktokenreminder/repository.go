package tickticktokenreminder

import (
	"context"
	"time"

	"github.com/prepin/tick-sync/internal/infra/sqlite/tickticktokens"
)

// TokenRepository stores TickTick token metadata and reminder idempotency markers.
type TokenRepository interface {
	Get(ctx context.Context) (tickticktokens.Token, error)
	MarkRefreshReminderCreated(ctx context.Context, accessToken string, taskID string, createdAt time.Time) error
}

//go:generate go tool mockgen -source=repository.go -destination=mocks/repository_mocks.go -package=mocks
