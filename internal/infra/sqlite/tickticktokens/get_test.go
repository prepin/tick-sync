package tickticktokens

import (
	"errors"
	"testing"
)

// Reports that TickTick has not been connected when no token has been saved.
func TestGetReportsMissingToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	_, err := repo.Get(t.Context())
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected missing token error, got %v", err)
	}
}

// Returns the bearer token value required by the TickTick API client.
func TestGetAccessTokenReturnsStoredAccessToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), tokenFixture()); err != nil {
		t.Fatalf("save ticktick token: %v", err)
	}

	got, err := repo.GetAccessToken(t.Context())
	if err != nil {
		t.Fatalf("get ticktick access token: %v", err)
	}
	if got != "access-1" {
		t.Fatalf("unexpected access token: %s", got)
	}
}

// Reads tokens that were saved before reminder columns existed and therefore contain NULL reminder fields.
func TestGetAllowsMissingReminderColumnsOnExistingToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), tokenFixture()); err != nil {
		t.Fatalf("save ticktick token: %v", err)
	}
	if _, err := repo.db.ExecContext(t.Context(), `UPDATE ticktick_tokens SET refresh_reminder_task_id = NULL, refresh_reminder_created_at = NULL`); err != nil {
		t.Fatalf("clear reminder columns: %v", err)
	}

	got, err := repo.Get(t.Context())
	if err != nil {
		t.Fatalf("get ticktick token: %v", err)
	}
	if got.RefreshReminderTaskID != "" {
		t.Fatalf("unexpected reminder task id: %s", got.RefreshReminderTaskID)
	}
}
