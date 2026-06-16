package googletasks

import (
	"context"

	"github.com/prepin/tick-sync/internal/application/googletasksync"
	"github.com/prepin/tick-sync/internal/config"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	tasksapi "google.golang.org/api/tasks/v1"
)

// Client is the Google Tasks API adapter.
type Client struct {
	service    *tasksapi.Service
	taskListID string
}

// New creates a Google Tasks client from config.
func New(ctx context.Context, cfg config.Config) (*Client, error) {
	var opts []option.ClientOption

	if cfg.GoogleAPIEndpoint != "" {
		opts = append(opts, option.WithEndpoint(cfg.GoogleAPIEndpoint), option.WithoutAuthentication())
	} else {
		oauthConfig := &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			Endpoint:     googleoauth.Endpoint,
			Scopes:       []string{tasksapi.TasksScope},
		}

		token := &oauth2.Token{
			AccessToken:  cfg.GoogleAccessToken,
			RefreshToken: cfg.GoogleRefreshToken,
			TokenType:    cfg.GoogleTokenType,
			Expiry:       cfg.GoogleTokenExpiry,
		}

		opts = append(opts, option.WithHTTPClient(oauthConfig.Client(ctx, token)))
	}

	service, err := tasksapi.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{service: service, taskListID: cfg.GoogleTaskListID}, nil
}

// ListUncompleted returns all uncompleted Google tasks in the configured task list.
func (c *Client) ListUncompleted(ctx context.Context) ([]googletasksync.GoogleTaskView, error) {
	var tasks []*tasksapi.Task
	pageToken := ""

	for {
		call := c.service.Tasks.List(c.taskListID).
			Context(ctx).
			ShowCompleted(false).
			ShowDeleted(false)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		result, err := call.Do()
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, result.Items...)
		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	mapped := make([]googletasksync.GoogleTaskView, len(tasks))
	for i, task := range tasks {
		mapped[i] = toGoogleTaskView(task)
	}

	return mapped, nil
}

// Complete marks the given Google task as completed.
func (c *Client) Complete(ctx context.Context, taskID string) error {
	_, err := c.service.Tasks.Patch(c.taskListID, taskID, &tasksapi.Task{Status: "completed"}).Context(ctx).Do()
	return err
}

// Delete removes the given Google task.
func (c *Client) Delete(ctx context.Context, taskID string) error {
	return c.service.Tasks.Delete(c.taskListID, taskID).Context(ctx).Do()
}
