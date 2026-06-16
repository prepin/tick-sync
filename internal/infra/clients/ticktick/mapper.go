package ticktick

import (
	"fmt"
	"time"

	"github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

func toCreateTaskRequest(input googletasksync.CreateTickTickTaskInput, projectID, timeZone string) (createTaskRequest, error) {
	dueDate, hasDueDate, err := formatDueDate(input.Due)
	if err != nil {
		return createTaskRequest{}, err
	}

	request := createTaskRequest{
		Title:     input.Title,
		ProjectID: projectID,
		Content:   input.Details,
		TimeZone:  timeZone,
	}

	if hasDueDate {
		request.DueDate = dueDate
		request.IsAllDay = new(true)
	}

	return request, nil
}

func toTickTickTaskView(resp createTaskResponse) googletasksync.TickTickTaskView {
	return googletasksync.TickTickTaskView{ID: resp.ID}
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
