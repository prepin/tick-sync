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

	_ "modernc.org/sqlite"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/infra/sqlite/migrate"
	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
)

// Shows the start page with Google and TickTick connect links before tokens are stored.
func TestIndexShowsConnectLinks(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Google Tasks is not connected") {
		t.Fatalf("expected missing google token status, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "TickTick is not connected") {
		t.Fatalf("expected missing token status, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "/google/auth") {
		t.Fatalf("expected google auth link, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "/ticktick/auth") {
		t.Fatalf("expected auth link, got %q", rec.Body.String())
	}
}

// Allows requests without credentials when HTTP basic auth is not configured.
func TestBasicAuthDisabledWhenPasswordIsEmpty(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{HTTPBasicAuthUsername: "tick-sync"}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

// Rejects requests without credentials when HTTP basic auth is configured.
func TestBasicAuthRejectsMissingCredentials(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{
		HTTPBasicAuthUsername: "tick-sync",
		HTTPBasicAuthPassword: "secret",
	}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("expected basic auth challenge")
	}
}

// Rejects requests with incorrect HTTP basic auth credentials.
func TestBasicAuthRejectsWrongCredentials(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{
		HTTPBasicAuthUsername: "tick-sync",
		HTTPBasicAuthPassword: "secret",
	}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("tick-sync", "wrong")
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

// Allows requests with matching HTTP basic auth credentials.
func TestBasicAuthAcceptsValidCredentials(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{
		HTTPBasicAuthUsername: "tick-sync",
		HTTPBasicAuthPassword: "secret",
	}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("tick-sync", "secret")
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

// Protects OAuth callbacks with HTTP basic auth when it is configured.
func TestBasicAuthProtectsOAuthCallback(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{
		HTTPBasicAuthUsername: "tick-sync",
		HTTPBasicAuthPassword: "secret",
	}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/google/callback?code=auth-code&state=state-1", nil)
	req.AddCookie(&http.Cookie{Name: "google_oauth_state", Value: "state-1"})
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

// Redirects the browser to Google authorization with offline access and forced consent.
func TestGoogleAuthRedirectsToProvider(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRedirectURL:  "http://localhost:8080/google/callback",
		GoogleAuthURL:      "https://google.example/oauth/authorize",
	}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/google/auth", nil)
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "https://google.example/oauth/authorize?") {
		t.Fatalf("unexpected redirect location: %s", location)
	}
	if rec.Result().Cookies()[0].Name != "google_oauth_state" {
		t.Fatalf("expected oauth state cookie, got %+v", rec.Result().Cookies())
	}
	for _, want := range []string{"client_id=client-id", "access_type=offline", "prompt=consent", "scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Ftasks"} {
		if !strings.Contains(location, want) {
			t.Fatalf("expected %s in redirect location: %s", want, location)
		}
	}
}

// Rejects a Google callback when the browser state does not match the saved OAuth state cookie.
func TestGoogleCallbackRejectsInvalidState(t *testing.T) {
	t.Parallel()
	h := newHandler(config.Config{}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/google/callback?code=auth-code&state=bad", nil)
	req.AddCookie(&http.Cookie{Name: "google_oauth_state", Value: "good"})
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

// Exchanges the authorization code and stores the returned Google refresh token in SQLite.
func TestGoogleCallbackStoresExchangedToken(t *testing.T) {
	t.Parallel()
	repo := newTestTokenRepo(t)
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("client_id") != "client-id" || r.Form.Get("client_secret") != "client-secret" {
			t.Fatalf("unexpected client credentials")
		}
		if r.Form.Get("code") != "auth-code" {
			t.Fatalf("unexpected code: %s", r.Form.Get("code"))
		}
		writeJSON(
			t,
			w,
			map[string]any{
				"access_token":  "access-1",
				"refresh_token": "refresh-1",
				"token_type":    "Bearer",
				"expires_in":    3600,
			},
		)
	}))
	t.Cleanup(tokenServer.Close)
	h := newHandler(config.Config{
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRedirectURL:  "http://localhost:8080/google/callback",
		GoogleTokenURL:     tokenServer.URL,
	}, repo)
	req := httptest.NewRequest(http.MethodGet, "/google/callback?code=auth-code&state=state-1", nil)
	req.AddCookie(&http.Cookie{Name: "google_oauth_state", Value: "state-1"})
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body %q", rec.Code, rec.Body.String())
	}
	token, err := repo.Get(t.Context(), oauthtokens.ProviderGoogle)
	if err != nil {
		t.Fatalf("get google token: %v", err)
	}
	if token.RefreshToken != "refresh-1" {
		t.Fatalf("unexpected refresh token: %s", token.RefreshToken)
	}
}

// Reports a useful error when Google does not return a refresh token.
func TestGoogleCallbackRejectsMissingRefreshToken(t *testing.T) {
	t.Parallel()
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, map[string]any{"access_token": "access-1", "token_type": "Bearer"})
	}))
	t.Cleanup(tokenServer.Close)
	h := newHandler(config.Config{
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRedirectURL:  "http://localhost:8080/google/callback",
		GoogleTokenURL:     tokenServer.URL,
	}, newTestTokenRepo(t))
	req := httptest.NewRequest(http.MethodGet, "/google/callback?code=auth-code&state=state-1", nil)
	req.AddCookie(&http.Cookie{Name: "google_oauth_state", Value: "state-1"})
	rec := httptest.NewRecorder()

	h.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "missing refresh_token") {
		t.Fatalf("expected missing refresh token error, got %q", rec.Body.String())
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
	token, err := repo.Get(t.Context(), oauthtokens.ProviderTickTick)
	if err != nil {
		t.Fatalf("get ticktick token: %v", err)
	}
	if token.AccessToken != "access-1" {
		t.Fatalf("unexpected access token: %s", token.AccessToken)
	}
}

// Creates an OAuth token repo with a fresh SQLite database for HTTP handler tests.
func newTestTokenRepo(t *testing.T) *oauthtokens.Repo {
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
	repo, err := oauthtokens.New(db)
	if err != nil {
		t.Fatalf("new oauth token repo: %v", err)
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
