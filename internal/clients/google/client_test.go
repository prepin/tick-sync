package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prepin/tick-sync/internal/usecases/googletasksync"
	"google.golang.org/api/option"
	googletasks "google.golang.org/api/tasks/v1"
)

func TestListUncompletedMapsGoogleTasksFromAPI(t *testing.T) {
	ctx := context.Background()
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

	want := []googletasksync.GoogleTask{
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

func TestCompletePatchesGoogleTaskStatus(t *testing.T) {
	ctx := context.Background()
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

func TestDeleteDeletesGoogleTask(t *testing.T) {
	ctx := context.Background()
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

func TestMapTaskMapsGoogleTaskFields(t *testing.T) {
	task := &googletasks.Task{
		Id:      "google-1",
		Title:   "Buy milk",
		Notes:   "Remember lactose-free",
		Status:  "needsAction",
		Due:     "2026-06-12T00:00:00.000Z",
		Updated: "2026-06-10T10:00:00.000Z",
	}

	got := mapTask(task)
	want := googletasksync.GoogleTask{
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

func TestMapTaskHandlesNilTask(t *testing.T) {
	got := mapTask(nil)
	if got != (googletasksync.GoogleTask{}) {
		t.Fatalf("unexpected mapped task: %+v", got)
	}
}

func newTestClient(t *testing.T, ctx context.Context, endpoint string) *Client {
	t.Helper()

	service, err := googletasks.NewService(
		ctx,
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("new google tasks service: %v", err)
	}

	return &Client{service: service, taskListID: "@default"}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil && !strings.Contains(err.Error(), "connection") {
		t.Fatalf("write json: %v", err)
	}
}
