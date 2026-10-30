package tickticktokenreminder

import (
	"context"
	"fmt"
	"time"

	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
	"github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

func (u *UseCase) createExpiryReminder(ctx context.Context, token oauthtokens.Token) error {
	if token.ExpiryReminderTaskID != "" {
		return nil
	}
	now := u.now().In(u.location)
	noon := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, u.location)
	if now.After(noon) {
		noon = noon.AddDate(0, 0, 1)
	}
	taskID, err := u.google.CreateTask(ctx, CreateGoogleTaskInput{
		Title: "Reconnect TickTick — token expired",
		Notes: fmt.Sprintf(
			"TickTick token expired at %s. Reconnect TickTick at http://localhost:8080/.\n"+
				"Intended due time: %s (%s). Google Tasks API stores the date only.\n%s",
			token.ExpiresAt.In(u.location).Format(timeLayout),
			noon.Format(timeLayout),
			u.location.String(),
			googletasksync.TickTickExpiryReminderMarker,
		),
		Due: noon,
	})
	if err != nil {
		return fmt.Errorf("create google reminder for expired ticktick token: %w", err)
	}
	// ponytail: API creation and SQLite recording are not atomic; add reconciliation if crash duplicates matter.
	if err := u.tokens.MarkExpiryReminderCreated(
		ctx, oauthtokens.ProviderTickTick, token.AccessToken, taskID, now,
	); err != nil {
		return fmt.Errorf("record ticktick token expiry reminder: %w", err)
	}
	return nil
}
