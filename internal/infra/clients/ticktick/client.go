// Package ticktick implements a client for the TickTick API.
package ticktick

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/prepin/tick-sync/internal/config"
)

const (
	defaultAPIBaseURL = "https://api.ticktick.com/open/v1"
	maxErrorBodyBytes = 4096
)

// Client is the TickTick HTTP API adapter.
type Client struct {
	httpClient    *http.Client
	tokenProvider AccessTokenProvider
	baseURL       string
	timeZone      string
	projectID     string
}

// AccessTokenProvider provides the current TickTick API bearer token.
type AccessTokenProvider interface {
	GetAccessToken(ctx context.Context, provider string) (string, error)
}

// New creates a TickTick client from config.
func New(cfg config.Config, tokenProvider AccessTokenProvider) (*Client, error) {
	if tokenProvider == nil {
		return nil, errors.New("ticktick client: token provider is nil")
	}

	apiBaseURL := cmp.Or(cfg.TickTickAPIBaseURL, defaultAPIBaseURL)

	parsedBaseURL, err := url.Parse(apiBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse ticktick api base url: %w", err)
	}
	if !parsedBaseURL.IsAbs() {
		return nil, errors.New("parse ticktick api base url: URL must be absolute")
	}

	return &Client{
		httpClient:    &http.Client{Timeout: cfg.HTTPClientTimeout},
		tokenProvider: tokenProvider,
		baseURL:       apiBaseURL,
		timeZone:      cfg.TZ,
		projectID:     cfg.TickTickProjectID,
	}, nil
}

func readErrorBody(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	if err != nil {
		return "<failed to read response body>"
	}

	return strings.TrimSpace(string(data))
}
