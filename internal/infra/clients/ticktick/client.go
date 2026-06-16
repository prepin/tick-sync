package ticktick

import (
	"cmp"
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
	defaultTimeZone   = "UTC"
	maxErrorBodyBytes = 4096
)

// Client is the TickTick HTTP API adapter.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	timeZone   string
	projectID  string
}

// Option configures the client.
type Option func(*Client)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// New creates a TickTick client from config.
func New(cfg config.Config, opts ...Option) (*Client, error) {
	if cfg.TickTickAccessToken == "" {
		return nil, errors.New("missing required environment variable: TICKTICK_ACCESS_TOKEN")
	}

	apiBaseURL := cmp.Or(cfg.TickTickAPIBaseURL, defaultAPIBaseURL)

	parsedBaseURL, err := url.Parse(apiBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse ticktick api base url: %w", err)
	}
	if !parsedBaseURL.IsAbs() {
		return nil, errors.New("parse ticktick api base url: URL must be absolute")
	}

	client := &Client{
		httpClient: http.DefaultClient,
		baseURL:    apiBaseURL,
		token:      cfg.TickTickAccessToken,
		timeZone:   cmp.Or(cfg.TickTickTimeZone, defaultTimeZone),
		projectID:  cfg.TickTickProjectID,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

func readErrorBody(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	if err != nil {
		return "<failed to read response body>"
	}

	return strings.TrimSpace(string(data))
}
