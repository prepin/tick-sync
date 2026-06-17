package tickticktokens

import "testing"

// Stores a TickTick token so future sync ticks can use the latest access token.
func TestSaveStoresToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), tokenFixture()); err != nil {
		t.Fatalf("save ticktick token: %v", err)
	}

	got, err := repo.Get(t.Context())
	if err != nil {
		t.Fatalf("get ticktick token: %v", err)
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

// Replaces the stored TickTick token when the OAuth flow is completed again.
func TestSaveReplacesExistingToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), tokenFixture()); err != nil {
		t.Fatalf("save initial ticktick token: %v", err)
	}
	updated := tokenFixture()
	updated.AccessToken = "access-2"
	if err := repo.Save(t.Context(), updated); err != nil {
		t.Fatalf("save updated ticktick token: %v", err)
	}

	got, err := repo.Get(t.Context())
	if err != nil {
		t.Fatalf("get ticktick token: %v", err)
	}
	if got.AccessToken != "access-2" {
		t.Fatalf("unexpected access token: %s", got.AccessToken)
	}
}
