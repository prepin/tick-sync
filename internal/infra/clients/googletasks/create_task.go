package googletasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	tasksapi "google.golang.org/api/tasks/v1"

	"github.com/prepin/tick-sync/internal/usecase/tickticktokenreminder"
)

// CreateTask creates a task in the configured Google task list and returns its ID.
func (c *Client) CreateTask(ctx context.Context, input tickticktokenreminder.CreateGoogleTaskInput) (string, error) {
	task := &tasksapi.Task{Title: input.Title, Notes: input.Notes}
	if !input.Due.IsZero() {
		// Google discards the time: encode the selected local date, not its UTC conversion.
		task.Due = time.Date(input.Due.Year(), input.Due.Month(), input.Due.Day(), 0, 0, 0, 0, time.UTC).
			Format(time.RFC3339)
	}
	created, err := c.service.Tasks.Insert(c.taskListID, task).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("create google task: %w", err)
	}
	if created.Id == "" {
		return "", errors.New("create google task: response missing task id")
	}
	return created.Id, nil
}
