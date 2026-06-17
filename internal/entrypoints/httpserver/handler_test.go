package httpserver

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/infra/sqlite/migrate"
	"github.com/prepin/tick-sync/internal/infra/sqlite/tickticktokens"
	_ "modernc.org/sqlite"
)

// Shows the start page with a TickTick connect link before any token is stored.
func TestIndexShowsTickTickConnectLink(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "TickTick is not connected") {
		t.Fatalf("expected missing token status, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "/ticktick/auth") {
		t.Fatalf("expected auth link, got %q", rec.Body.String())
	}
}

// Redirects the browser to TickTick authorization with the configured OAuth details.
func TestTickTickAuthRedirectsToProvider(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{
		TickTickClientID:     "client-id",
		TickTickClientSecret: "client-secret",
		TickTickRedirectURL:  "http://localhost:8080/ticktick/callback",
		TickTickAuthURL:      "https://ticktick.example/oauth/authorize",
	}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/ticktick/auth", nil)
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "https://ticktick.example/oauth/authorize?") {
		t.Fatalf("unexpected redirect location: %s", location)
	}
	if rec.Result().Cookies()[0].Name != "ticktick_oauth_state" {
		t.Fatalf("expected oauth state cookie, got %+v", rec.Result().Cookies())
	}
	if !strings.Contains(location, "client_id=client-id") {
		t.Fatalf("expected client id in redirect location: %s", location)
	}
	if !strings.Contains(location, "scope=tasks%3Aread+tasks%3Awrite") {
		t.Fatalf("expected scope in redirect location: %s", location)
	}
}

// Rejects a callback when the browser state does not match the saved OAuth state cookie.
func TestTickTickCallbackRejectsInvalidState(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/ticktick/callback?code=auth-code&state=bad", nil)
	req.AddCookie(&http.Cookie{Name: "ticktick_oauth_state", Value: "good"})
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

// Exchanges the authorization code and stores the returned TickTick token in SQLite.
func TestTickTickCallbackStoresExchangedToken(t *testing.T) {
	t.Parallel()
	repo := newTestTokenRepo(t)
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "client-id" || password != "client-secret" {
			t.Fatalf("unexpected basic auth: %s %s %v", username, password, ok)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("code") != "auth-code" {
			t.Fatalf("unexpected code: %s", r.Form.Get("code"))
		}
		writeJSON(t, w, map[string]any{"access_token": "access-1", "token_type": "bearer", "expires_in": 3600})
	}))
	t.Cleanup(tokenServer.Close)
	h := newHandler(config.Config{
		TickTickClientID:     "client-id",
		TickTickClientSecret: "client-secret",
		TickTickRedirectURL:  "http://localhost:8080/ticktick/callback",
		TickTickTokenURL:     tokenServer.URL,
	}, repo)
	req := httptest.NewRequest(http.MethodGet, "/ticktick/callback?code=auth-code&state=state-1", nil)
	req.AddCookie(&http.Cookie{Name: "ticktick_oauth_state", Value: "state-1"})
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body %q", rec.Code, rec.Body.String())
	}
	token, err := repo.Get(t.Context())
	if err != nil {
		t.Fatalf("get ticktick token: %v", err)
	}
	if token.AccessToken != "access-1" {
		t.Fatalf("unexpected access token: %s", token.AccessToken)
	}
}

// Creates a TickTick token repo with a fresh SQLite database for HTTP handler tests.
func newTestTokenRepo(t *testing.T) *tickticktokens.Repo {
	t.Helper()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "tick-sync.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := migrate.Up(t.Context(), db); err != nil {
		db.Close()
		t.Fatalf("run sqlite migrations: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close sqlite db: %v", err)
		}
	})
	repo, err := tickticktokens.New(db)
	if err != nil {
		t.Fatalf("new ticktick token repo: %v", err)
	}
	return repo
}

// Encodes the value as JSON and writes it to the response writer.
func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}
}

// Verifies that generated OAuth state values are URL-safe and non-empty.
func TestRandomStateCreatesURLSafeValue(t *testing.T) {
	t.Parallel()

	state, err := randomState()
	if err != nil {
		t.Fatalf("random state: %v", err)
	}
	if state == "" {
		t.Fatal("expected state")
	}
	if _, err := base64.RawURLEncoding.DecodeString(state); err != nil {
		t.Fatalf("expected URL-safe base64 state: %v", err)
	}
}
