package config

import (
	"strings"
	"testing"
	"time"
)

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
	if cfg.GooglePostSyncAction != "complete" {
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

func TestValidateDoesNotRequireTickTickCredentialsYet(t *testing.T) {
	cfg := Config{
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRefreshToken: "refresh-token",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}
}

func TestParseTokenExpiryDefaultsToExpiredTime(t *testing.T) {
	expiry, err := parseTokenExpiry("")
	if err != nil {
		t.Fatalf("parse token expiry: %v", err)
	}

	if !expiry.Before(time.Now()) {
		t.Fatalf("expected default expiry to be in the past, got %s", expiry)
	}
}

func TestParsePollIntervalDefaultsToFiveMinutes(t *testing.T) {
	interval, err := parsePollInterval("")
	if err != nil {
		t.Fatalf("parse poll interval: %v", err)
	}
	if interval != 5*time.Minute {
		t.Fatalf("unexpected interval: %s", interval)
	}
}

func TestParsePollIntervalParsesDuration(t *testing.T) {
	interval, err := parsePollInterval("30s")
	if err != nil {
		t.Fatalf("parse poll interval: %v", err)
	}
	if interval != 30*time.Second {
		t.Fatalf("unexpected interval: %s", interval)
	}
}

func TestParsePollIntervalRejectsInvalidDuration(t *testing.T) {
	_, err := parsePollInterval("soon")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParsePollIntervalRejectsNonPositiveDuration(t *testing.T) {
	for _, value := range []string{"0s", "-1m"} {
		_, err := parsePollInterval(value)
		if err == nil {
			t.Fatalf("expected error for %q", value)
		}
	}
}

func TestParseTokenExpiryParsesRFC3339(t *testing.T) {
	expiry, err := parseTokenExpiry("2026-06-10T12:00:00Z")
	if err != nil {
		t.Fatalf("parse token expiry: %v", err)
	}

	if expiry.Format(time.RFC3339) != "2026-06-10T12:00:00Z" {
		t.Fatalf("unexpected expiry: %s", expiry.Format(time.RFC3339))
	}
}

func TestParseTokenExpiryRejectsInvalidValue(t *testing.T) {
	_, err := parseTokenExpiry("not-a-date")
	if err == nil {
		t.Fatal("expected error")
	}
}
