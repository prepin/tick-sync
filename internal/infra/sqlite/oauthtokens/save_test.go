package oauthtokens

import "testing"

// Stores an OAuth token so provider clients can use the latest access token.
func TestSaveStoresToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), ProviderTickTick, tokenFixture()); err != nil {
		t.Fatalf("save oauth token: %v", err)
	}

	got, err := repo.Get(t.Context(), ProviderTickTick)
	if err != nil {
		t.Fatalf("get oauth token: %v", err)
	}
	if got.AccessToken != "access-1" {
		t.Fatalf("unexpected access token: %s", got.AccessToken)
	}
	if got.RefreshToken != "refresh-1" {
		t.Fatalf("unexpected refresh token: %s", got.RefreshToken)
	}
	if !got.ExpiresAt.Equal(tokenFixture().ExpiresAt) {
		t.Fatalf("unexpected expiry: %s", got.ExpiresAt)
	}
}

// Preserves the refresh reminder marker when the same token is saved again.
func TestSavePreservesReminderForSameToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), ProviderTickTick, tokenFixture()); err != nil {
		t.Fatalf("save initial oauth token: %v", err)
	}
	if err := repo.MarkRefreshReminderCreated(
		t.Context(),
		ProviderTickTick,
		"access-1",
		"task-1",
		tokenFixture().UpdatedAt,
	); err != nil {
		t.Fatalf("mark refresh reminder created: %v", err)
	}
	if err := repo.Save(t.Context(), ProviderTickTick, tokenFixture()); err != nil {
		t.Fatalf("save updated oauth token: %v", err)
	}

	got, err := repo.Get(t.Context(), ProviderTickTick)
	if err != nil {
		t.Fatalf("get oauth token: %v", err)
	}
	if got.RefreshReminderTaskID != "task-1" {
		t.Fatalf("unexpected reminder task id: %s", got.RefreshReminderTaskID)
	}
}

// Clears the refresh reminder marker when OAuth stores a new access token.
func TestSaveClearsReminderForNewToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), ProviderTickTick, tokenFixture()); err != nil {
		t.Fatalf("save initial oauth token: %v", err)
	}
	if err := repo.MarkRefreshReminderCreated(
		t.Context(),
		ProviderTickTick,
		"access-1",
		"task-1",
		tokenFixture().UpdatedAt,
	); err != nil {
		t.Fatalf("mark refresh reminder created: %v", err)
	}
	updated := tokenFixture()
	updated.AccessToken = "access-2"
	if err := repo.Save(t.Context(), ProviderTickTick, updated); err != nil {
		t.Fatalf("save updated oauth token: %v", err)
	}

	got, err := repo.Get(t.Context(), ProviderTickTick)
	if err != nil {
		t.Fatalf("get oauth token: %v", err)
	}
	if got.AccessToken != "access-2" {
		t.Fatalf("unexpected access token: %s", got.AccessToken)
	}
	if got.RefreshReminderTaskID != "" {
		t.Fatalf("expected reminder marker to be cleared, got %s", got.RefreshReminderTaskID)
	}
}
