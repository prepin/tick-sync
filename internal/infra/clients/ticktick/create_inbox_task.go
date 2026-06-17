package ticktick

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
	"github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

// CreateInboxTask creates a TickTick inbox task from the usecase input.
func (c *Client) CreateInboxTask(
	ctx context.Context,
	input googletasksync.CreateTickTickTaskInput,
) (googletasksync.TickTickTaskView, error) {
	token, err := c.tokenProvider.GetAccessToken(ctx, oauthtokens.ProviderTickTick)
	if err != nil {
		return googletasksync.TickTickTaskView{}, err
	}

	requestBody, err := toCreateTaskRequest(input, c.projectID, c.timeZone)
	if err != nil {
		return googletasksync.TickTickTaskView{}, err
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return googletasksync.TickTickTaskView{}, fmt.Errorf("marshal ticktick create task request: %w", err)
	}

	requestURL, err := url.JoinPath(c.baseURL, "task")
	if err != nil {
		return googletasksync.TickTickTaskView{}, fmt.Errorf("build ticktick request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return googletasksync.TickTickTaskView{}, fmt.Errorf("create ticktick request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return googletasksync.TickTickTaskView{}, fmt.Errorf("send ticktick create task request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return googletasksync.TickTickTaskView{}, fmt.Errorf(
			"create ticktick task: status %d: %s",
			resp.StatusCode,
			readErrorBody(resp.Body),
		)
	}

	var created createTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return googletasksync.TickTickTaskView{}, fmt.Errorf("decode ticktick create task response: %w", err)
	}
	if created.ID == "" {
		return googletasksync.TickTickTaskView{}, errors.New("decode ticktick create task response: missing id")
	}

	return toTickTickTaskView(created), nil
}
