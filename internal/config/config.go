package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleAccessToken  string
	GoogleRefreshToken string
	GoogleTokenType    string
	GoogleTokenExpiry  time.Time
	GoogleTaskListID   string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		GoogleClientID:     strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleClientSecret: strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		GoogleAccessToken:  strings.TrimSpace(os.Getenv("GOOGLE_ACCESS_TOKEN")),
		GoogleRefreshToken: strings.TrimSpace(os.Getenv("GOOGLE_REFRESH_TOKEN")),
		GoogleTokenType:    strings.TrimSpace(os.Getenv("GOOGLE_TOKEN_TYPE")),
		GoogleTaskListID:   strings.TrimSpace(os.Getenv("GOOGLE_TASKLIST_ID")),
	}

	if cfg.GoogleTokenType == "" {
		cfg.GoogleTokenType = "Bearer"
	}
	if cfg.GoogleTaskListID == "" {
		cfg.GoogleTaskListID = "@default"
	}

	expiry, err := parseTokenExpiry(os.Getenv("GOOGLE_TOKEN_EXPIRY"))
	if err != nil {
		return Config{}, err
	}
	cfg.GoogleTokenExpiry = expiry

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var missing []string

	if c.GoogleClientID == "" {
		missing = append(missing, "GOOGLE_CLIENT_ID")
	}
	if c.GoogleClientSecret == "" {
		missing = append(missing, "GOOGLE_CLIENT_SECRET")
	}
	if c.GoogleRefreshToken == "" {
		missing = append(missing, "GOOGLE_REFRESH_TOKEN")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func parseTokenExpiry(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().Add(-time.Hour), nil
	}

	expiry, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, errors.New("GOOGLE_TOKEN_EXPIRY must be an RFC3339 timestamp, for example 2026-06-10T12:00:00Z")
	}

	return expiry, nil
}
