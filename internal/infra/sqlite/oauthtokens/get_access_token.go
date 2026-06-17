package oauthtokens

import "context"

// GetAccessToken returns only the bearer token value required by API clients.
func (r *Repo) GetAccessToken(ctx context.Context, provider string) (string, error) {
	token, err := r.Get(ctx, provider)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
