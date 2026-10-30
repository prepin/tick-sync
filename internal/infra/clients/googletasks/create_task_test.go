package googletasks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tasksapi "google.golang.org/api/tasks/v1"

	"github.com/prepin/tick-sync/internal/usecase/tickticktokenreminder"
)

// Creates reminders in the configured list with the selected local date, without shifting it to UTC.
func TestCreateTaskStoresTitleNotesAndLocalDueDate(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tasks/v1/lists/reminders/tasks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var task tasksapi.Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			t.Errorf("decode task: %v", err)
		}
		if task.Title != "Reconnect TickTick" || task.Notes != "Reconnect instructions" {
			t.Errorf("unexpected task: %+v", task)
		}
		if task.Due != "2026-06-18T00:00:00Z" {
			t.Errorf("unexpected due date: %s", task.Due)
		}
		writeJSON(t, w, map[string]string{"id": "reminder-1"})
	}))
	t.Cleanup(server.Close)
	client := newTestClient(t, t.Context(), server.URL+"/")
	client.taskListID = "reminders"
	id, err := client.CreateTask(t.Context(), tickticktokenreminder.CreateGoogleTaskInput{
		Title: "Reconnect TickTick",
		Notes: "Reconnect instructions",
		Due:   time.Date(2026, 6, 18, 12, 0, 0, 0, time.FixedZone("UTC+14", 14*60*60)),
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if id != "reminder-1" {
		t.Fatalf("unexpected task ID: %s", id)
	}
}

// Leaves the due date unset when no date was requested.
func TestCreateTaskOmitsUnspecifiedDueDate(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var task map[string]any
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			t.Errorf("decode task: %v", err)
		}
		if _, found := task["due"]; found {
			t.Errorf("unexpected due date: %v", task["due"])
		}
		writeJSON(t, w, map[string]string{"id": "reminder-1"})
	}))
	t.Cleanup(server.Close)
	_, err := newTestClient(t, t.Context(), server.URL+"/").CreateTask(
		t.Context(), tickticktokenreminder.CreateGoogleTaskInput{Title: "Reconnect TickTick"},
	)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
}

// Reports failed or unusable creation responses without returning a reminder ID.
func TestCreateTaskReportsUnsuccessfulCreation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "Google authentication rejected", status: http.StatusUnauthorized, body: `{"error":{"message":"unauthorized"}}`},
		{name: "Google temporarily unavailable", status: http.StatusServiceUnavailable, body: `{"error":{"message":"unavailable"}}`},
		{name: "created task ID missing", status: http.StatusOK, body: `{}`},
		{name: "invalid response", status: http.StatusOK, body: `not JSON`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)
			id, err := newTestClient(t, t.Context(), server.URL+"/").CreateTask(
				t.Context(), tickticktokenreminder.CreateGoogleTaskInput{Title: "Reconnect TickTick"},
			)
			if err == nil || id != "" {
				t.Fatalf("expected creation failure without task ID, got %q, %v", id, err)
			}
		})
	}
}
