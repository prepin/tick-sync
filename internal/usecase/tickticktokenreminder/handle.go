package tickticktokenreminder

import (
	"context"
	"errors"
	"fmt"

	"github.com/prepin/tick-sync/internal/infra/sqlite/tickticktokens"
	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

// Handle creates a single medium-priority reminder task when the stored token expires within two weeks.
func (u *UseCase) Handle(ctx context.Context) error {
	token, err := u.tokens.Get(ctx)
	if errors.Is(err, tickticktokens.ErrTokenNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get ticktick token: %w", err)
	}
	if !u.shouldCreateReminder(token) {
		return nil
	}

	created, err := u.ticktick.CreateInboxTask(ctx, googletasksync.CreateTickTickTaskInput{
		Title:    "Refresh TickTick token",
		Details:  fmt.Sprintf("TickTick token expires at %s. Reconnect TickTick at http://localhost:8080/.", token.ExpiresAt.Format(timeLayout)),
		Priority: mediumPriority,
	})
	if err != nil {
		return fmt.Errorf("create ticktick token refresh reminder: %w", err)
	}

	if err := u.tokens.MarkRefreshReminderCreated(ctx, token.AccessToken, created.ID, u.now()); err != nil {
		return fmt.Errorf("record ticktick token refresh reminder: %w", err)
	}

	return nil
}

const timeLayout = "2006-01-02 15:04:05 MST"

func (u *UseCase) shouldCreateReminder(token tickticktokens.Token) bool {
	if token.ExpiresAt.IsZero() || token.RefreshReminderTaskID != "" {
		return false
	}
	return !token.ExpiresAt.After(u.now().Add(refreshReminderWindow))
}
