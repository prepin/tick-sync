package google

import (
	"context"

	"github.com/prepin/tick-sync/internal/config"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	googletasks "google.golang.org/api/tasks/v1"
)

type Client struct {
	service *googletasks.Service
}

func New(ctx context.Context, cfg config.Config) (*Client, error) {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		Endpoint:     googleoauth.Endpoint,
		Scopes:       []string{googletasks.TasksScope},
	}

	token := &oauth2.Token{
		AccessToken:  cfg.GoogleAccessToken,
		RefreshToken: cfg.GoogleRefreshToken,
		TokenType:    cfg.GoogleTokenType,
		Expiry:       cfg.GoogleTokenExpiry,
	}

	service, err := googletasks.NewService(ctx, option.WithHTTPClient(oauthConfig.Client(ctx, token)))
	if err != nil {
		return nil, err
	}

	return &Client{service: service}, nil
}

func (c *Client) ListUncompletedTasks(ctx context.Context, taskListID string) ([]*googletasks.Task, error) {
	var tasks []*googletasks.Task
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
