package googletasks

import (
	"context"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	tasksapi "google.golang.org/api/tasks/v1"
)

type Client struct {
	service    *tasksapi.Service
	taskListID string
}

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

func (c *Client) ListUncompleted(ctx context.Context) ([]googletasksync.GoogleTask, error) {
	tasks, err := c.ListUncompletedTasks(ctx, c.taskListID)
	if err != nil {
		return nil, err
	}

	mapped := make([]googletasksync.GoogleTask, 0, len(tasks))
	for _, task := range tasks {
		mapped = append(mapped, mapTask(task))
	}

	return mapped, nil
}

func (c *Client) Complete(ctx context.Context, taskID string) error {
	_, err := c.service.Tasks.Patch(c.taskListID, taskID, &tasksapi.Task{Status: "completed"}).Context(ctx).Do()
	return err
}

func (c *Client) Delete(ctx context.Context, taskID string) error {
	return c.service.Tasks.Delete(c.taskListID, taskID).Context(ctx).Do()
}

func (c *Client) ListUncompletedTasks(ctx context.Context, taskListID string) ([]*tasksapi.Task, error) {
	var tasks []*tasksapi.Task
	pageToken := ""

	for {
		call := c.service.Tasks.List(taskListID).
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

	return tasks, nil
}

func mapTask(task *tasksapi.Task) googletasksync.GoogleTask {
	if task == nil {
		return googletasksync.GoogleTask{}
	}

	return googletasksync.GoogleTask{
		ID:      task.Id,
		Title:   task.Title,
		Notes:   task.Notes,
		Status:  task.Status,
		Due:     task.Due,
		Updated: task.Updated,
	}
}
