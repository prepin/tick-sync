package httpserver

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prepin/tick-sync/internal/config"
	"github.com/prepin/tick-sync/internal/infra/sqlite/oauthtokens"
)

const (
	tickTickScope = "tasks:read tasks:write"
	googleScope   = "https://www.googleapis.com/auth/tasks"
)

var indexTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>tick-sync</title></head>
<body>
<main>
  <h1>tick-sync</h1>
  <h2>Google Tasks</h2>
  <p>{{.GoogleStatus}}</p>
  <p><a href="/google/auth">Connect Google Tasks</a></p>
  <h2>TickTick</h2>
  <p>{{.TickTickStatus}}</p>
  <p><a href="/ticktick/auth">Connect TickTick</a></p>
</main>
</body>
</html>`))

type handler struct {
	cfg        config.Config
	tokens     TokenStore
	httpClient *http.Client
}

func newHandler(cfg config.Config, tokens TokenStore) *handler {
	return &handler{cfg: cfg, tokens: tokens, httpClient: http.DefaultClient}
}

func (h *handler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.index)
	mux.HandleFunc("GET /google/auth", h.googleAuth)
	mux.HandleFunc("GET /google/callback", h.googleCallback)
	mux.HandleFunc("GET /ticktick/auth", h.tickTickAuth)
	mux.HandleFunc("GET /ticktick/callback", h.tickTickCallback)
	return mux
}

func (h *handler) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := map[string]string{
		"GoogleStatus":   statusText(r.Context(), h.tokens, oauthtokens.ProviderGoogle, "Google Tasks"),
		"TickTickStatus": statusText(r.Context(), h.tokens, oauthtokens.ProviderTickTick, "TickTick"),
	}
	if err := indexTemplate.Execute(w, data); err != nil {
		http.Error(w, "render page", http.StatusInternalServerError)
	}
}

func (h *handler) googleAuth(w http.ResponseWriter, r *http.Request) {
	if h.cfg.GoogleClientID == "" || h.cfg.GoogleClientSecret == "" {
		http.Error(w, "Google OAuth is not configured", http.StatusInternalServerError)
		return
	}

	state, err := randomState()
	if err != nil {
		http.Error(w, "create oauth state", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "google_oauth_state",
		Value:    state,
		Path:     "/google",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	values := url.Values{}
	values.Set("client_id", h.cfg.GoogleClientID)
	values.Set("redirect_uri", h.cfg.GoogleRedirectURL)
	values.Set("response_type", "code")
	values.Set("scope", googleScope)
	values.Set("state", state)
	values.Set("access_type", "offline")
	values.Set("prompt", "consent")
	http.Redirect(w, r, h.cfg.GoogleAuthURL+"?"+values.Encode(), http.StatusFound)
}

func (h *handler) googleCallback(w http.ResponseWriter, r *http.Request) {
	if err := h.validateState(r, "google_oauth_state"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := h.exchangeGoogleCode(r.Context(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if err := h.tokens.Save(r.Context(), oauthtokens.ProviderGoogle, token); err != nil {
		http.Error(w, "save google token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "google_oauth_state", Value: "", Path: "/google", MaxAge: -1, HttpOnly: true})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!doctype html><title>Google connected</title><p>Google Tasks connected. You can close this page.</p><p><a href=\"/\">Back</a></p>"))
}

func (h *handler) tickTickAuth(w http.ResponseWriter, r *http.Request) {
	if h.cfg.TickTickClientID == "" || h.cfg.TickTickClientSecret == "" {
		http.Error(w, "TickTick OAuth is not configured", http.StatusInternalServerError)
		return
	}

	state, err := randomState()
	if err != nil {
		http.Error(w, "create oauth state", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "ticktick_oauth_state",
		Value:    state,
		Path:     "/ticktick",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	values := url.Values{}
	values.Set("client_id", h.cfg.TickTickClientID)
	values.Set("redirect_uri", h.cfg.TickTickRedirectURL)
	values.Set("response_type", "code")
	values.Set("scope", tickTickScope)
	values.Set("state", state)
	http.Redirect(w, r, h.cfg.TickTickAuthURL+"?"+values.Encode(), http.StatusFound)
}

func (h *handler) tickTickCallback(w http.ResponseWriter, r *http.Request) {
	if err := h.validateState(r, "ticktick_oauth_state"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := h.exchangeTickTickCode(r.Context(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if err := h.tokens.Save(r.Context(), oauthtokens.ProviderTickTick, token); err != nil {
		http.Error(w, "save ticktick token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "ticktick_oauth_state", Value: "", Path: "/ticktick", MaxAge: -1, HttpOnly: true})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!doctype html><title>TickTick connected</title><p>TickTick connected. You can close this page.</p><p><a href=\"/\">Back</a></p>"))
}

func (h *handler) validateState(r *http.Request, cookieName string) error {
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(cookieName)
	if err != nil || state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		return fmt.Errorf("invalid oauth state")
	}
	return nil
}

func (h *handler) exchangeTickTickCode(ctx context.Context, code string) (oauthtokens.Token, error) {
	values := url.Values{}
	values.Set("code", code)
	values.Set("grant_type", "authorization_code")
	values.Set("scope", tickTickScope)
	values.Set("redirect_uri", h.cfg.TickTickRedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.cfg.TickTickTokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return oauthtokens.Token{}, fmt.Errorf("create ticktick token request: %w", err)
	}
	req.SetBasicAuth(h.cfg.TickTickClientID, h.cfg.TickTickClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return oauthtokens.Token{}, fmt.Errorf("exchange ticktick authorization code: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return oauthtokens.Token{}, fmt.Errorf("exchange ticktick authorization code: status %d", resp.StatusCode)
	}

	var body struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return oauthtokens.Token{}, fmt.Errorf("decode ticktick token response: %w", err)
	}
	if body.AccessToken == "" {
		return oauthtokens.Token{}, fmt.Errorf("decode ticktick token response: missing access_token")
	}
	if body.TokenType == "" {
		body.TokenType = "bearer"
	}

	token := oauthtokens.Token{
		AccessToken:  body.AccessToken,
		TokenType:    body.TokenType,
		Scope:        body.Scope,
		RefreshToken: body.RefreshToken,
		UpdatedAt:    time.Now(),
	}
	if body.ExpiresIn > 0 {
		token.ExpiresAt = token.UpdatedAt.Add(time.Duration(body.ExpiresIn) * time.Second)
	}
	return token, nil
}

func (h *handler) exchangeGoogleCode(ctx context.Context, code string) (oauthtokens.Token, error) {
	values := url.Values{}
	values.Set("code", code)
	values.Set("client_id", h.cfg.GoogleClientID)
	values.Set("client_secret", h.cfg.GoogleClientSecret)
	values.Set("grant_type", "authorization_code")
	values.Set("redirect_uri", h.cfg.GoogleRedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.cfg.GoogleTokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return oauthtokens.Token{}, fmt.Errorf("create google token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return oauthtokens.Token{}, fmt.Errorf("exchange google authorization code: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return oauthtokens.Token{}, fmt.Errorf("exchange google authorization code: status %d", resp.StatusCode)
	}

	var body struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return oauthtokens.Token{}, fmt.Errorf("decode google token response: %w", err)
	}
	if body.AccessToken == "" {
		return oauthtokens.Token{}, fmt.Errorf("decode google token response: missing access_token")
	}
	if body.RefreshToken == "" {
		return oauthtokens.Token{}, fmt.Errorf("decode google token response: missing refresh_token; reconnect and approve consent")
	}
	if body.TokenType == "" {
		body.TokenType = "Bearer"
	}

	token := oauthtokens.Token{
		AccessToken:  body.AccessToken,
		TokenType:    body.TokenType,
		Scope:        body.Scope,
		RefreshToken: body.RefreshToken,
		UpdatedAt:    time.Now(),
	}
	if body.ExpiresIn > 0 {
		token.ExpiresAt = token.UpdatedAt.Add(time.Duration(body.ExpiresIn) * time.Second)
	}
	return token, nil
}

func randomState() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
