package ticktick

import (
	"cmp"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prepin/tick-sync/internal/config"
)

const (
	defaultAPIBaseURL = "https://api.ticktick.com/open/v1"
	defaultTimeZone   = "UTC"
	maxErrorBodyBytes = 4096
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	timeZone   string
	projectID  string
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func New(cfg config.Config, opts ...Option) (*Client, error) {
	if cfg.TickTickAccessToken == "" {
		return nil, fmt.Errorf("missing required environment variable: TICKTICK_ACCESS_TOKEN")
	}

	apiBaseURL := cmp.Or(cfg.TickTickAPIBaseURL, defaultAPIBaseURL)

	parsedBaseURL, err := url.Parse(apiBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse ticktick api base url: %w", err)
	}
	if !parsedBaseURL.IsAbs() {
		return nil, fmt.Errorf("parse ticktick api base url: URL must be absolute")
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

func formatDueDate(value string) (string, bool, error) {
	if value == "" {
		return "", false, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", false, fmt.Errorf("parse google due date %q: %w", value, err)
	}

	return parsed.Format("2006-01-02T15:04:05-0700"), true, nil
}

func readErrorBody(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	if err != nil {
		return "<failed to read response body>"
	}

	return strings.TrimSpace(string(data))
}
