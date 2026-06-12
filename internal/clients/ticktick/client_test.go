package ticktick

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
)

// Does not create a client when the TickTick access token is not provided.
func TestNewRequiresAccessToken(t *testing.T) {
	t.Parallel()
	_, err := New(config.Config{TickTickAPIBaseURL: "https://example.com"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// Applies default API base URL and time zone when the config values are empty.
func TestNewAppliesDefaults(t *testing.T) {
	t.Parallel()
	client, err := New(config.Config{TickTickAccessToken: "ticktick-token"})
	if err != nil {
		t.Fatalf("new ticktick client: %v", err)
	}

	if client.baseURL.String() != defaultAPIBaseURL+"/" {
		t.Fatalf("unexpected base url: %s", client.baseURL.String())
	}
	if client.timeZone != defaultTimeZone {
		t.Fatalf("unexpected timezone: %s", client.timeZone)
	}
}

// Does not create a client when the API base URL is a relative path instead of absolute.
func TestNewRejectsRelativeBaseURL(t *testing.T) {
	t.Parallel()
	_, err := New(config.Config{
		TickTickAccessToken: "ticktick-token",
		TickTickAPIBaseURL:  "/open/v1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// Posts a task to TickTick without a projectId and returns the created task ID.
func TestCreateInboxTaskPostsTaskWithoutProjectID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/task" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer ticktick-token" {
			t.Fatalf("unexpected authorization header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := body["projectId"]; ok {
			t.Fatalf("expected projectId to be omitted, got %+v", body)
		}
		assertRequestField(t, body, "title", "Buy milk")
		assertRequestField(t, body, "content", "Remember lactose-free")
		assertRequestField(t, body, "dueDate", "2026-06-12T00:00:00+0000")
		assertRequestField(t, body, "timeZone", "UTC")
		if body["isAllDay"] != true {
			t.Fatalf("expected isAllDay true, got %+v", body["isAllDay"])
		}

		writeJSON(t, w, map[string]string{"id": "ticktick-1"})
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, "")
	created, err := client.CreateInboxTask(ctx, googletasksync.TickTickTaskInput{
		Title:   "Buy milk",
		Details: "Remember lactose-free",
		Due:     "2026-06-12T00:00:00.000Z",
	})
	if err != nil {
		t.Fatalf("create inbox task: %v", err)
	}
	if created.ID != "ticktick-1" {
		t.Fatalf("unexpected task id: %s", created.ID)
	}
}

// Includes the projectId field in the request body when the client is configured with a project ID.
func TestCreateInboxTaskIncludesProjectIDWhenConfigured(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		assertRequestField(t, body, "projectId", "project-1")
		writeJSON(t, w, map[string]string{"id": "ticktick-1"})
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, "project-1")
	if _, err := client.CreateInboxTask(ctx, googletasksync.TickTickTaskInput{Title: "Buy milk"}); err != nil {
		t.Fatalf("create inbox task: %v", err)
	}
}

// Omits the dueDate and isAllDay fields when the task input has no due date.
func TestCreateInboxTaskOmitsDueDateWhenMissing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := body["dueDate"]; ok {
			t.Fatalf("expected dueDate to be omitted, got %+v", body)
		}
		if _, ok := body["isAllDay"]; ok {
			t.Fatalf("expected isAllDay to be omitted, got %+v", body)
		}
		writeJSON(t, w, map[string]string{"id": "ticktick-1"})
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, "")
	if _, err := client.CreateInboxTask(ctx, googletasksync.TickTickTaskInput{Title: "Buy milk"}); err != nil {
		t.Fatalf("create inbox task: %v", err)
	}
}

// Does not create an inbox task when the TickTick API responds with a 4xx error and body.
func TestCreateInboxTaskReturnsNon2xxErrorWithBody(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"projectId required"}`, http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, "")
	_, err := client.CreateInboxTask(ctx, googletasksync.TickTickTaskInput{Title: "Buy milk"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status 400") || !strings.Contains(err.Error(), "projectId required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Does not create an inbox task when the API response body is not valid JSON.
func TestCreateInboxTaskReturnsInvalidJSONError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, "")
	_, err := client.CreateInboxTask(ctx, googletasksync.TickTickTaskInput{Title: "Buy milk"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "decode ticktick create task response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Does not create an inbox task when the API response is missing the task ID.
func TestCreateInboxTaskReturnsMissingIDError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]string{"title": "Buy milk"})
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL, "")
	_, err := client.CreateInboxTask(ctx, googletasksync.TickTickTaskInput{Title: "Buy milk"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Does not create an inbox task when the due date string cannot be parsed.
func TestCreateInboxTaskReturnsInvalidDueDateError(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, "https://example.com", "")
	_, err := client.CreateInboxTask(context.Background(), googletasksync.TickTickTaskInput{
		Title: "Buy milk",
		Due:   "tomorrow",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse google due date") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Converts an RFC3339Nano due date string into the TickTick API date format with UTC timezone suffix.
func TestFormatDueDate(t *testing.T) {
	t.Parallel()
	got, ok, err := formatDueDate("2026-06-12T00:00:00.000Z")
	if err != nil {
		t.Fatalf("format due date: %v", err)
	}
	if !ok {
		t.Fatal("expected due date")
	}
	if got != "2026-06-12T00:00:00+0000" {
		t.Fatalf("unexpected due date: %s", got)
	}
}

// Returns a fallback error message string when the HTTP error response body cannot be read.
func TestReadErrorBodyReturnsFallbackMessageWhenReadFails(t *testing.T) {
	t.Parallel()
	got := readErrorBody(&faultyReader{})
	if got != "<failed to read response body>" {
		t.Fatalf("unexpected error body: %q", got)
	}
}

// faultyReader implements io.Reader to simulate a read failure for testing readErrorBody on stdlib io.ReadCloser.
type faultyReader struct{}

func (f *faultyReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read failed")
}

// Creates a TickTick client configured for the given mock server URL and optional project ID.
func newTestClient(t *testing.T, baseURL string, projectID string) *Client {
	t.Helper()

	client, err := New(config.Config{
		TickTickAccessToken: "ticktick-token",
		TickTickAPIBaseURL:  baseURL,
		TickTickTimeZone:    "UTC",
		TickTickProjectID:   projectID,
	}, WithHTTPClient(http.DefaultClient))
	if err != nil {
		t.Fatalf("new ticktick client: %v", err)
	}

	return client
}

// Asserts that the given body map contains a string value at the named key matching the expected value.
func assertRequestField(t *testing.T, body map[string]any, name string, want string) {
	t.Helper()

	got, ok := body[name].(string)
	if !ok {
		t.Fatalf("expected %s to be a string, got %+v", name, body[name])
	}
	if got != want {
		t.Fatalf("unexpected %s: got %q, want %q", name, got, want)
	}
}

// Encodes the value as JSON and writes it to the response writer, failing the test if encoding fails.
func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}
}
