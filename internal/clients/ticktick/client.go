package ticktick

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
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

	apiBaseURL := cfg.TickTickAPIBaseURL
	if apiBaseURL == "" {
		apiBaseURL = defaultAPIBaseURL
	}

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
		timeZone:   cfg.TickTickTimeZone,
		projectID:  cfg.TickTickProjectID,
	}

	if client.timeZone == "" {
		client.timeZone = defaultTimeZone
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

func (c *Client) CreateInboxTask(ctx context.Context, task googletasksync.TickTickTaskInput) (googletasksync.TickTickTask, error) {
	requestBody, err := c.createTaskRequest(task)
	if err != nil {
		return googletasksync.TickTickTask{}, err
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return googletasksync.TickTickTask{}, fmt.Errorf("marshal ticktick create task request: %w", err)
	}

	requestURL, err := url.JoinPath(c.baseURL, "task")
	if err != nil {
		return googletasksync.TickTickTask{}, fmt.Errorf("build ticktick request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return googletasksync.TickTickTask{}, fmt.Errorf("create ticktick request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return googletasksync.TickTickTask{}, fmt.Errorf("send ticktick create task request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return googletasksync.TickTickTask{}, fmt.Errorf("create ticktick task: status %d: %s", resp.StatusCode, readErrorBody(resp.Body))
	}

	var created createTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return googletasksync.TickTickTask{}, fmt.Errorf("decode ticktick create task response: %w", err)
	}
	if created.ID == "" {
		return googletasksync.TickTickTask{}, fmt.Errorf("decode ticktick create task response: missing id")
	}

	return googletasksync.TickTickTask{ID: created.ID}, nil
}

func (c *Client) createTaskRequest(task googletasksync.TickTickTaskInput) (createTaskRequest, error) {
	dueDate, hasDueDate, err := formatDueDate(task.Due)
	if err != nil {
		return createTaskRequest{}, err
	}

	request := createTaskRequest{
		Title:     task.Title,
		ProjectID: c.projectID,
		Content:   task.Details,
		TimeZone:  c.timeZone,
	}

	if hasDueDate {
		request.DueDate = dueDate
		request.IsAllDay = boolPtr(true)
	}

	return request, nil
}

type createTaskRequest struct {
	Title     string `json:"title"`
	ProjectID string `json:"projectId,omitempty"`
	Content   string `json:"content,omitempty"`
	DueDate   string `json:"dueDate,omitempty"`
	TimeZone  string `json:"timeZone,omitempty"`
	IsAllDay  *bool  `json:"isAllDay,omitempty"`
}

type createTaskResponse struct {
	ID string `json:"id"`
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

func boolPtr(value bool) *bool {
	return &value
}

func readErrorBody(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes))
	if err != nil {
		return "<failed to read response body>"
	}

	return strings.TrimSpace(string(data))
}
