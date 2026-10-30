package oauthtokens

import (
	"errors"
	"testing"
)

// Stores the Google reminder separately from the pre-expiry TickTick reminder.
func TestMarkExpiryReminderCreatedStoresSeparateMarker(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	token := tokenFixture()
	token.RefreshReminderTaskID = "ticktick-reminder"
	if err := repo.Save(t.Context(), ProviderTickTick, token); err != nil {
		t.Fatalf("save token: %v", err)
	}
	if err := repo.MarkExpiryReminderCreated(
		t.Context(), ProviderTickTick, token.AccessToken, "google-reminder", token.UpdatedAt,
	); err != nil {
		t.Fatalf("mark expiry reminder: %v", err)
	}
	got, err := repo.Get(t.Context(), ProviderTickTick)
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	if got.ExpiryReminderTaskID != "google-reminder" || !got.ExpiryReminderCreatedAt.Equal(token.UpdatedAt) {
		t.Fatalf("unexpected expiry reminder: %+v", got)
	}
	if got.RefreshReminderTaskID != "ticktick-reminder" {
		t.Fatalf("pre-expiry reminder was overwritten: %s", got.RefreshReminderTaskID)
	}
}

// Rejects a stale-token update without suppressing reminders for the replacement connection.
func TestMarkExpiryReminderCreatedDoesNotMarkReplacementToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	token := tokenFixture()
	if err := repo.Save(t.Context(), ProviderTickTick, token); err != nil {
		t.Fatalf("save token: %v", err)
	}
	err := repo.MarkExpiryReminderCreated(
		t.Context(),
		ProviderTickTick,
		"old-token",
		"google-reminder",
		token.UpdatedAt,
	)
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected stale token rejection, got %v", err)
	}
	got, err := repo.Get(t.Context(), ProviderTickTick)
	if err != nil {
		t.Fatalf("read token: %v", err)
	}
	if got.ExpiryReminderTaskID != "" {
		t.Fatalf("replacement token incorrectly marked: %s", got.ExpiryReminderTaskID)
	}
}

// Reports storage failure when reminder delivery cannot be recorded.
func TestMarkExpiryReminderCreatedReportsUnavailableStorage(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	if err := repo.db.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}
	err := repo.MarkExpiryReminderCreated(
		t.Context(),
		ProviderTickTick,
		"access-1",
		"google-reminder",
		tokenFixture().UpdatedAt,
	)
	if err == nil {
		t.Fatal("expected storage failure")
	}
}
