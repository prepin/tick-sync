package config

import (
	"strings"
	"testing"
	"time"
)

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

func TestParseTokenExpiryDefaultsToExpiredTime(t *testing.T) {
	expiry, err := parseTokenExpiry("")
	if err != nil {
		t.Fatalf("parse token expiry: %v", err)
	}

	if !expiry.Before(time.Now()) {
		t.Fatalf("expected default expiry to be in the past, got %s", expiry)
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
