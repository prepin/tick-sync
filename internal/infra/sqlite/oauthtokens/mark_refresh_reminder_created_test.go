package oauthtokens

import (
	"errors"
	"testing"
)

// Stores the provider reminder task id so the same token is not reminded twice.
func TestMarkRefreshReminderCreatedStoresTaskID(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), ProviderTickTick, tokenFixture()); err != nil {
		t.Fatalf("save oauth token: %v", err)
	}
	if err := repo.MarkRefreshReminderCreated(t.Context(), ProviderTickTick, "access-1", "task-1", tokenFixture().UpdatedAt); err != nil {
		t.Fatalf("mark refresh reminder created: %v", err)
	}

	got, err := repo.Get(t.Context(), ProviderTickTick)
	if err != nil {
		t.Fatalf("get oauth token: %v", err)
	}
	if got.RefreshReminderTaskID != "task-1" {
		t.Fatalf("unexpected reminder task id: %s", got.RefreshReminderTaskID)
	}
	if !got.RefreshReminderCreatedAt.Equal(tokenFixture().UpdatedAt) {
		t.Fatalf("unexpected reminder created at: %s", got.RefreshReminderCreatedAt)
	}
}

// Reports that no token was updated when the reminder marker targets an old access token.
func TestMarkRefreshReminderCreatedReportsStaleToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), ProviderTickTick, tokenFixture()); err != nil {
		t.Fatalf("save oauth token: %v", err)
	}
	if err := repo.MarkRefreshReminderCreated(t.Context(), ProviderTickTick, "old-token", "task-1", tokenFixture().UpdatedAt); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected missing token error, got %v", err)
	}
}
