package ticktick

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
)

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
