package oauthtokens

import (
	"errors"
	"testing"
)

// Reports that a provider has not been connected when no token has been saved.
func TestGetReportsMissingToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	_, err := repo.Get(t.Context(), ProviderGoogle)
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected missing token error, got %v", err)
	}
}

// Reads tokens that contain NULL reminder fields.
func TestGetAllowsMissingReminderColumnsOnExistingToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), ProviderTickTick, tokenFixture()); err != nil {
		t.Fatalf("save oauth token: %v", err)
	}
	if _, err := repo.db.ExecContext(t.Context(), `UPDATE oauth_tokens SET refresh_reminder_task_id = NULL, refresh_reminder_created_at = NULL`); err != nil {
		t.Fatalf("clear reminder columns: %v", err)
	}

	got, err := repo.Get(t.Context(), ProviderTickTick)
	if err != nil {
		t.Fatalf("get oauth token: %v", err)
	}
	if got.RefreshReminderTaskID != "" {
		t.Fatalf("unexpected reminder task id: %s", got.RefreshReminderTaskID)
	}
}
