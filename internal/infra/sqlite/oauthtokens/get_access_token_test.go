package oauthtokens

import "testing"

// Returns the bearer token value required by provider API clients.
func TestGetAccessTokenReturnsStoredAccessToken(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)

	if err := repo.Save(t.Context(), ProviderTickTick, tokenFixture()); err != nil {
		t.Fatalf("save oauth token: %v", err)
	}

	got, err := repo.GetAccessToken(t.Context(), ProviderTickTick)
	if err != nil {
		t.Fatalf("get oauth access token: %v", err)
	}
	if got != "access-1" {
		t.Fatalf("unexpected access token: %s", got)
	}
}
