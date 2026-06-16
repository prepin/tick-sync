package googletasks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prepin/tick-sync/internal/application/googletasksync"
	"google.golang.org/api/option"
	tasksapi "google.golang.org/api/tasks/v1"
)

// Maps the Google Tasks API response into domain tasks with all expected fields.
func TestListUncompletedMapsGoogleTasksFromAPI(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/tasks/v1/lists/@default/tasks" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("showCompleted") != "false" {
			t.Fatalf("unexpected showCompleted: %s", r.URL.Query().Get("showCompleted"))
		}
		if r.URL.Query().Get("showDeleted") != "false" {
			t.Fatalf("unexpected showDeleted: %s", r.URL.Query().Get("showDeleted"))
		}

		writeJSON(t, w, map[string]any{
			"items": []map[string]string{
				{
					"id":      "google-1",
					"title":   "Buy milk",
					"notes":   "Remember lactose-free",
					"status":  "needsAction",
					"due":     "2026-06-12T00:00:00.000Z",
					"updated": "2026-06-10T10:00:00.000Z",
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, ctx, server.URL+"/")

	got, err := client.ListUncompleted(ctx)
	if err != nil {
		t.Fatalf("list uncompleted: %v", err)
	}

	want := []googletasksync.GoogleTaskView{
		{
			ID:      "google-1",
			Title:   "Buy milk",
			Notes:   "Remember lactose-free",
			Status:  "needsAction",
			Due:     "2026-06-12T00:00:00.000Z",
			Updated: "2026-06-10T10:00:00.000Z",
		},
	}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("unexpected tasks: got %+v, want %+v", got, want)
	}
}

// Patches the Google Task status to "completed" via the Tasks API.
func TestCompletePatchesGoogleTaskStatus(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/tasks/v1/lists/@default/tasks/google-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Status != "completed" {
			t.Fatalf("unexpected status: %s", body.Status)
		}

		writeJSON(t, w, map[string]string{"id": "google-1", "status": "completed"})
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, ctx, server.URL+"/")
	if err := client.Complete(ctx, "google-1"); err != nil {
		t.Fatalf("complete task: %v", err)
	}
}

// Reports an error when the PATCH request to complete a task responds with a non-2xx status.
func TestCompleteReportsErrorOnNon2xxResponse(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "task not found", http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, ctx, server.URL+"/")
	if err := client.Complete(ctx, "google-1"); err == nil {
		t.Fatal("expected error from complete")
	}
}

// Sends a DELETE request to the Tasks API for the specified task.
func TestDeleteDeletesGoogleTask(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/tasks/v1/lists/@default/tasks/google-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, ctx, server.URL+"/")
	if err := client.Delete(ctx, "google-1"); err != nil {
		t.Fatalf("delete task: %v", err)
	}
}

// Reports an error when the DELETE request responds with a non-2xx status.
func TestDeleteReportsErrorOnNon2xxResponse(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, ctx, server.URL+"/")
	if err := client.Delete(ctx, "google-1"); err == nil {
		t.Fatal("expected error from delete")
	}
}

// Collects all tasks from every page when the API paginates the response.
func TestListUncompletedCollectsTasksAcrossPages(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	page := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		switch page {
		case 1:
			if r.URL.Query().Get("pageToken") != "" {
				t.Fatalf("unexpected pageToken on first request: %s", r.URL.Query().Get("pageToken"))
			}
			writeJSON(t, w, map[string]any{
				"items": []map[string]string{
					{"id": "google-1", "title": "First task"},
				},
				"nextPageToken": "page-2-token",
			})
		case 2:
			if r.URL.Query().Get("pageToken") != "page-2-token" {
				t.Fatalf("unexpected pageToken on second request: %s", r.URL.Query().Get("pageToken"))
			}
			writeJSON(t, w, map[string]any{
				"items": []map[string]string{
					{"id": "google-2", "title": "Second task"},
				},
			})
		default:
			t.Fatalf("unexpected page request: %d", page)
		}
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, ctx, server.URL+"/")
	got, err := client.ListUncompleted(ctx)
	if err != nil {
		t.Fatalf("list uncompleted: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got))
	}
	if got[0].ID != "google-1" || got[0].Title != "First task" {
		t.Fatalf("unexpected first task: %+v", got[0])
	}
	if got[1].ID != "google-2" || got[1].Title != "Second task" {
		t.Fatalf("unexpected second task: %+v", got[1])
	}
}

// Does not return any tasks when the API responds with a non-2xx status code.
func TestListUncompletedReturnsErrorOnNon2xxResponse(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, ctx, server.URL+"/")
	_, err := client.ListUncompleted(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

// Maps a non-nil Google Tasks API task into a GoogleTaskView with all fields populated.
func TestToGoogleTaskViewMapsGoogleTaskFields(t *testing.T) {
	t.Parallel()
	task := &tasksapi.Task{
		Id:      "google-1",
		Title:   "Buy milk",
		Notes:   "Remember lactose-free",
		Status:  "needsAction",
		Due:     "2026-06-12T00:00:00.000Z",
		Updated: "2026-06-10T10:00:00.000Z",
	}

	got := toGoogleTaskView(task)
	want := googletasksync.GoogleTaskView{
		ID:      "google-1",
		Title:   "Buy milk",
		Notes:   "Remember lactose-free",
		Status:  "needsAction",
		Due:     "2026-06-12T00:00:00.000Z",
		Updated: "2026-06-10T10:00:00.000Z",
	}

	if got != want {
		t.Fatalf("unexpected mapped task: got %+v, want %+v", got, want)
	}
}

// Returns an empty GoogleTaskView struct when a nil API task is provided.
func TestToGoogleTaskViewReturnsEmptyTaskForNilInput(t *testing.T) {
	t.Parallel()
	got := toGoogleTaskView(nil)
	if got != (googletasksync.GoogleTaskView{}) {
		t.Fatalf("unexpected mapped task: %+v", got)
	}
}

// Creates a Google Tasks client using the test endpoint without authentication.
func newTestClient(t *testing.T, ctx context.Context, endpoint string) *Client {
	t.Helper()

	service, err := tasksapi.NewService(
		ctx,
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("new google tasks service: %v", err)
	}

	return &Client{service: service, taskListID: "@default"}
}

// Encodes the value as JSON and writes it to the response writer, failing the test if encoding fails.
func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil && !strings.Contains(err.Error(), "connection") {
		t.Fatalf("write json: %v", err)
	}
}
