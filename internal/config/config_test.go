package config

import (
	"strings"
	"testing"
	"time"

	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

// Loads config with all defaults applied when environment variables are empty.
func TestLoadAppliesOperationalDefaults(t *testing.T) {
	t.Setenv("DB_PATH", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("GOOGLE_POST_SYNC_ACTION", "")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_TOKEN_TYPE", "")
	t.Setenv("GOOGLE_TOKEN_EXPIRY", "")
	t.Setenv("GOOGLE_TASKLIST_ID", "")
	t.Setenv("GOOGLE_REDIRECT_URL", "")
	t.Setenv("GOOGLE_AUTH_URL", "")
	t.Setenv("GOOGLE_TOKEN_URL", "")
	t.Setenv("POLL_INTERVAL", "")
	t.Setenv("TICKTICK_REMINDER_INTERVAL", "")
	t.Setenv("GOOGLE_TODAY_IMPORT_DELAY", "")
	t.Setenv("TZ", "")
	t.Setenv("TICKTICK_API_BASE_URL", "")
	t.Setenv("TICKTICK_REDIRECT_URL", "")
	t.Setenv("TICKTICK_AUTH_URL", "")
	t.Setenv("TICKTICK_TOKEN_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.DBPath != "./tick-sync.db" {
		t.Fatalf("unexpected db path: %s", cfg.DBPath)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("unexpected http addr: %s", cfg.HTTPAddr)
	}
	if cfg.HTTPBasicAuthUsername != "tick-sync" {
		t.Fatalf("unexpected http basic auth username: %s", cfg.HTTPBasicAuthUsername)
	}
	if cfg.HTTPBasicAuthPassword != "" {
		t.Fatalf("unexpected http basic auth password: %s", cfg.HTTPBasicAuthPassword)
	}
	if cfg.GooglePostSyncAction != googletasksync.PostSyncActionComplete {
		t.Fatalf("unexpected post sync action: %s", cfg.GooglePostSyncAction)
	}
	if cfg.PollInterval != 5*time.Minute {
		t.Fatalf("unexpected poll interval: %s", cfg.PollInterval)
	}
	if cfg.TickTickReminderInterval != 24*time.Hour {
		t.Fatalf("unexpected ticktick reminder interval: %s", cfg.TickTickReminderInterval)
	}
	if cfg.GoogleTodayImportDelay {
		t.Fatal("expected today import delay to be disabled by default")
	}
	if cfg.Location == nil {
		t.Fatal("expected system local timezone to be configured")
	}
	if cfg.GoogleTokenType != "Bearer" {
		t.Fatalf("unexpected google token type: %s", cfg.GoogleTokenType)
	}
	if cfg.GoogleTaskListID != "@default" {
		t.Fatalf("unexpected google task list id: %s", cfg.GoogleTaskListID)
	}
	if cfg.GoogleRedirectURL != "http://localhost:8080/google/callback" {
		t.Fatalf("unexpected google redirect url: %s", cfg.GoogleRedirectURL)
	}
	if cfg.GoogleAuthURL != "https://accounts.google.com/o/oauth2/v2/auth" {
		t.Fatalf("unexpected google auth url: %s", cfg.GoogleAuthURL)
	}
	if cfg.GoogleTokenURL != "https://oauth2.googleapis.com/token" {
		t.Fatalf("unexpected google token url: %s", cfg.GoogleTokenURL)
	}
	if cfg.TickTickAPIBaseURL != "https://api.ticktick.com/open/v1" {
		t.Fatalf("unexpected ticktick api base url: %s", cfg.TickTickAPIBaseURL)
	}
	if cfg.TickTickRedirectURL != "http://localhost:8080/ticktick/callback" {
		t.Fatalf("unexpected ticktick redirect url: %s", cfg.TickTickRedirectURL)
	}
	if cfg.TickTickAuthURL != "https://ticktick.com/oauth/authorize" {
		t.Fatalf("unexpected ticktick auth url: %s", cfg.TickTickAuthURL)
	}
	if cfg.TickTickTokenURL != "https://ticktick.com/oauth/token" {
		t.Fatalf("unexpected ticktick token url: %s", cfg.TickTickTokenURL)
	}
}

// Fails validation when any of the required Google OAuth client environment variables are missing.
func TestValidateRequiresGoogleOAuthValues(t *testing.T) {
	t.Parallel()
	cfg := Config{}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	message := err.Error()
	for _, name := range []string{
		"GOOGLE_CLIENT_ID",
		"GOOGLE_CLIENT_SECRET",
	} {
		if !strings.Contains(message, name) {
			t.Fatalf("expected error to mention %s, got %q", name, message)
		}
	}
}

// Passes validation when only Google OAuth client credentials are provided.
func TestValidateAllowsClientCredentialsOnly(t *testing.T) {
	t.Parallel()
	cfg := Config{
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}
}

// Loads a time in the past as the default token expiry when no value is configured.
func TestLoadDefaultsTokenExpiryToExpiredTime(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_TOKEN_EXPIRY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !cfg.GoogleTokenExpiry.Before(time.Now()) {
		t.Fatalf("expected default expiry to be in the past, got %s", cfg.GoogleTokenExpiry)
	}
}

// Loads a valid RFC3339 timestamp into the token expiry.
func TestLoadParsesTokenExpiryRFC3339(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_TOKEN_EXPIRY", "2026-06-10T12:00:00Z")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.GoogleTokenExpiry.Format(time.RFC3339) != "2026-06-10T12:00:00Z" {
		t.Fatalf("unexpected expiry: %s", cfg.GoogleTokenExpiry.Format(time.RFC3339))
	}
}

// Reports an error when GOOGLE_TOKEN_EXPIRY is not a valid RFC3339 timestamp.
func TestLoadRejectsInvalidTokenExpiry(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_TOKEN_EXPIRY", "not-a-date")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "GoogleTokenExpiry") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Loads a custom duration into the configured poll interval.
func TestLoadParsesPollIntervalDuration(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("POLL_INTERVAL", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Fatalf("unexpected interval: %s", cfg.PollInterval)
	}
}

// Loads a custom duration into the configured TickTick reminder interval.
func TestLoadParsesTickTickReminderIntervalDuration(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("TICKTICK_REMINDER_INTERVAL", "12h")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.TickTickReminderInterval != 12*time.Hour {
		t.Fatalf("unexpected interval: %s", cfg.TickTickReminderInterval)
	}
}

// Loads the conventional TZ environment variable as the application timezone.
func TestLoadParsesTZLocation(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("TZ", "Europe/Warsaw")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.TZ != "Europe/Warsaw" {
		t.Fatalf("unexpected TZ: %s", cfg.TZ)
	}
	if cfg.Location.String() != "Europe/Warsaw" {
		t.Fatalf("unexpected location: %s", cfg.Location)
	}
}

// Reports an error when TZ is not a known IANA timezone name.
func TestLoadRejectsInvalidTZ(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("TZ", "Mars/Base")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "TZ") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Loads the today-import delay toggle from the environment.
func TestLoadParsesTodayImportDelayToggle(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_TODAY_IMPORT_DELAY", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.GoogleTodayImportDelay {
		t.Fatal("expected today import delay to be enabled")
	}
}

// Reports an error when POLL_INTERVAL is not a valid duration.
func TestLoadRejectsInvalidPollInterval(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("POLL_INTERVAL", "soon")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "PollInterval") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Reports an error when POLL_INTERVAL is zero or negative.
func TestLoadRejectsNonPositivePollInterval(t *testing.T) {
	for _, value := range []string{"0s", "-1m"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("GOOGLE_CLIENT_ID", "client-id")
			t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
			t.Setenv("POLL_INTERVAL", value)

			_, err := Load()
			if err == nil {
				t.Fatalf("expected error for %q", value)
			}
			if !strings.Contains(err.Error(), "POLL_INTERVAL") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// Reports an error when TICKTICK_REMINDER_INTERVAL is zero or negative.
func TestLoadRejectsNonPositiveTickTickReminderInterval(t *testing.T) {
	for _, value := range []string{"0s", "-1m"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("GOOGLE_CLIENT_ID", "client-id")
			t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
			t.Setenv("TICKTICK_REMINDER_INTERVAL", value)

			_, err := Load()
			if err == nil {
				t.Fatalf("expected error for %q", value)
			}
			if !strings.Contains(err.Error(), "TICKTICK_REMINDER_INTERVAL") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// Reports an error when GOOGLE_POST_SYNC_ACTION is not "complete" or "delete".
func TestLoadRejectsInvalidPostSyncAction(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_POST_SYNC_ACTION", "archive")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "GOOGLE_POST_SYNC_ACTION") {
		t.Fatalf("unexpected error: %v", err)
	}
}
