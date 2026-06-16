package config

import (
	"strings"
	"testing"
	"time"

	googletasksync "github.com/prepin/tick-sync/internal/application/googletasksync"
)

// Loads config with all defaults applied when environment variables are empty.
func TestLoadAppliesOperationalDefaults(t *testing.T) {
	t.Setenv("DB_PATH", "")
	t.Setenv("GOOGLE_POST_SYNC_ACTION", "")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_REFRESH_TOKEN", "refresh-token")
	t.Setenv("GOOGLE_ACCESS_TOKEN", "")
	t.Setenv("GOOGLE_TOKEN_TYPE", "")
	t.Setenv("GOOGLE_TOKEN_EXPIRY", "")
	t.Setenv("GOOGLE_TASKLIST_ID", "")
	t.Setenv("TICKTICK_API_BASE_URL", "")
	t.Setenv("TICKTICK_TIME_ZONE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.DBPath != "./tick-sync.db" {
		t.Fatalf("unexpected db path: %s", cfg.DBPath)
	}
	if cfg.GooglePostSyncAction != googletasksync.PostSyncActionComplete {
		t.Fatalf("unexpected post sync action: %s", cfg.GooglePostSyncAction)
	}
	if cfg.PollInterval != 5*time.Minute {
		t.Fatalf("unexpected poll interval: %s", cfg.PollInterval)
	}
	if cfg.GoogleTokenType != "Bearer" {
		t.Fatalf("unexpected google token type: %s", cfg.GoogleTokenType)
	}
	if cfg.GoogleTaskListID != "@default" {
		t.Fatalf("unexpected google task list id: %s", cfg.GoogleTaskListID)
	}
	if cfg.TickTickAPIBaseURL != "https://api.ticktick.com/open/v1" {
		t.Fatalf("unexpected ticktick api base url: %s", cfg.TickTickAPIBaseURL)
	}
	if cfg.TickTickTimeZone != "UTC" {
		t.Fatalf("unexpected ticktick timezone: %s", cfg.TickTickTimeZone)
	}
}

// Fails validation when any of the required Google OAuth environment variables are missing.
func TestValidateRequiresGoogleOAuthValues(t *testing.T) {
	cfg := Config{}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	message := err.Error()
	for _, name := range []string{
		"GOOGLE_CLIENT_ID",
		"GOOGLE_CLIENT_SECRET",
		"GOOGLE_REFRESH_TOKEN",
	} {
		if !strings.Contains(message, name) {
			t.Fatalf("expected error to mention %s, got %q", name, message)
		}
	}
}

// Passes validation when only Google OAuth credentials (no access token) are provided.
func TestValidateAllowsRefreshTokenOnly(t *testing.T) {
	cfg := Config{
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRefreshToken: "refresh-token",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}
}

// Returns a time in the past as the default token expiry when no value is configured.
func TestParseTokenExpiryDefaultsToExpiredTime(t *testing.T) {
	expiry, err := parseTokenExpiry("")
	if err != nil {
		t.Fatalf("parse token expiry: %v", err)
	}

	if !expiry.Before(time.Now()) {
		t.Fatalf("expected default expiry to be in the past, got %s", expiry)
	}
}

// Parses a valid RFC3339 timestamp into the token expiry.
func TestParseTokenExpiryParsesRFC3339(t *testing.T) {
	expiry, err := parseTokenExpiry("2026-06-10T12:00:00Z")
	if err != nil {
		t.Fatalf("parse token expiry: %v", err)
	}

	if expiry.Format(time.RFC3339) != "2026-06-10T12:00:00Z" {
		t.Fatalf("unexpected expiry: %s", expiry.Format(time.RFC3339))
	}
}

// Reports an error when the token expiry value is not a valid RFC3339 timestamp.
func TestParseTokenExpiryReportsErrorForInvalidValue(t *testing.T) {
	_, err := parseTokenExpiry("not-a-date")
	if err == nil {
		t.Fatal("expected error")
	}
}

// Returns five minutes as the default poll interval when no value is configured.
func TestParsePollIntervalDefaultsToFiveMinutes(t *testing.T) {
	interval, err := parsePollInterval("")
	if err != nil {
		t.Fatalf("parse poll interval: %v", err)
	}
	if interval != 5*time.Minute {
		t.Fatalf("unexpected interval: %s", interval)
	}
}

// Parses a valid duration string into the configured poll interval.
func TestParsePollIntervalParsesDuration(t *testing.T) {
	interval, err := parsePollInterval("30s")
	if err != nil {
		t.Fatalf("parse poll interval: %v", err)
	}
	if interval != 30*time.Second {
		t.Fatalf("unexpected interval: %s", interval)
	}
}

// Reports an error when the poll interval string is not a valid duration.
func TestParsePollIntervalReportsErrorForInvalidDurationString(t *testing.T) {
	_, err := parsePollInterval("soon")
	if err == nil {
		t.Fatal("expected error")
	}
}

// Reports an error when the poll interval is zero or negative.
func TestParsePollIntervalReportsErrorForNonPositiveDuration(t *testing.T) {
	for _, value := range []string{"0s", "-1m"} {
		_, err := parsePollInterval(value)
		if err == nil {
			t.Fatalf("expected error for %q", value)
		}
	}
}

// Reports an error when GOOGLE_POST_SYNC_ACTION is not "complete" or "delete".
func TestLoadRejectsInvalidPostSyncAction(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_REFRESH_TOKEN", "refresh-token")
	t.Setenv("GOOGLE_POST_SYNC_ACTION", "archive")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "GOOGLE_POST_SYNC_ACTION") {
		t.Fatalf("unexpected error: %v", err)
	}
}
