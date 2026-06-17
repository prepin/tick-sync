// Package googletasks implements a client for the Google Tasks API.
package googletasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	tasksapi "google.golang.org/api/tasks/v1"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
	"github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

// TokenStore provides persisted Google OAuth tokens.
type TokenStore interface {
	Get(ctx context.Context, provider string) (oauthtokens.Token, error)
	Save(ctx context.Context, provider string, token oauthtokens.Token) error
}

// Client is the Google Tasks API adapter.
type Client struct {
	service    *tasksapi.Service
	taskListID string
}

// New creates a Google Tasks client from config.
func New(ctx context.Context, cfg config.Config, tokens TokenStore) (*Client, error) {
	var opts []option.ClientOption

	if cfg.GoogleAPIEndpoint != "" {
		opts = append(opts, option.WithEndpoint(cfg.GoogleAPIEndpoint), option.WithoutAuthentication())
	} else {
		if tokens == nil {
			return nil, errors.New("google tasks client: token store is nil")
		}

		oauthConfig := &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			Endpoint:     googleoauth.Endpoint,
			Scopes:       []string{tasksapi.TasksScope},
		}

		opts = append(opts, option.WithHTTPClient(oauth2.NewClient(ctx, &dbTokenSource{
			config: oauthConfig,
			tokens: tokens,
		})))
	}

	service, err := tasksapi.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{service: service, taskListID: cfg.GoogleTaskListID}, nil
}

type dbTokenSource struct {
	config *oauth2.Config
	tokens TokenStore
}

func (s *dbTokenSource) Token() (*oauth2.Token, error) {
	ctx := context.Background()
	stored, err := s.tokens.Get(ctx, oauthtokens.ProviderGoogle)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  stored.AccessToken,
		TokenType:    stored.TokenType,
		RefreshToken: stored.RefreshToken,
		Expiry:       stored.ExpiresAt,
	}
	if token.Valid() {
		return token, nil
	}
	if token.RefreshToken == "" {
		return nil, oauthtokens.ErrTokenNotFound
	}

	refreshed, err := s.config.TokenSource(ctx, token).Token()
	if err != nil {
		return nil, fmt.Errorf("refresh google oauth token: %w", err)
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = token.RefreshToken
	}
	updated := oauthtokens.Token{
		AccessToken:  refreshed.AccessToken,
		TokenType:    refreshed.TokenType,
		RefreshToken: refreshed.RefreshToken,
		ExpiresAt:    refreshed.Expiry,
		UpdatedAt:    time.Now(),
	}
	if err := s.tokens.Save(ctx, oauthtokens.ProviderGoogle, updated); err != nil {
		return nil, fmt.Errorf("save refreshed google oauth token: %w", err)
	}

	return refreshed, nil
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
