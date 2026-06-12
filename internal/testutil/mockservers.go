package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// StartMockServers creates HTTP test servers for the Google Tasks and TickTick APIs
// that respond to a one-task sync scenario (list, create, complete).
func StartMockServers(t *testing.T) (googleServer, ticktickServer *httptest.Server) {
	t.Helper()

	googleServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]string{
					{"id": "g1", "title": "Buy milk"},
				},
			})
		case http.MethodPatch:
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id":     "g1",
				"status": "completed",
			})
		default:
			t.Errorf("unexpected Google method: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(googleServer.Close)

	ticktickServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "t1"})
	}))
	t.Cleanup(ticktickServer.Close)

	return googleServer, ticktickServer
}
