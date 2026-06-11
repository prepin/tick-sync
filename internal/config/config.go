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
	DBPath               string
	GooglePostSyncAction string
	PollInterval         time.Duration

	GoogleClientID     string
	GoogleClientSecret string
	GoogleAccessToken  string
	GoogleRefreshToken string
	GoogleTokenType    string
	GoogleTokenExpiry  time.Time
	GoogleTaskListID   string

	TickTickAccessToken string
	TickTickAPIBaseURL  string
	TickTickTimeZone    string
	TickTickProjectID   string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		DBPath:               strings.TrimSpace(os.Getenv("DB_PATH")),
		GooglePostSyncAction: strings.TrimSpace(os.Getenv("GOOGLE_POST_SYNC_ACTION")),
		GoogleClientID:       strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		GoogleClientSecret:   strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
		GoogleAccessToken:    strings.TrimSpace(os.Getenv("GOOGLE_ACCESS_TOKEN")),
		GoogleRefreshToken:   strings.TrimSpace(os.Getenv("GOOGLE_REFRESH_TOKEN")),
		GoogleTokenType:      strings.TrimSpace(os.Getenv("GOOGLE_TOKEN_TYPE")),
		GoogleTaskListID:     strings.TrimSpace(os.Getenv("GOOGLE_TASKLIST_ID")),
		TickTickAccessToken:  strings.TrimSpace(os.Getenv("TICKTICK_ACCESS_TOKEN")),
		TickTickAPIBaseURL:   strings.TrimSpace(os.Getenv("TICKTICK_API_BASE_URL")),
		TickTickTimeZone:     strings.TrimSpace(os.Getenv("TICKTICK_TIME_ZONE")),
		TickTickProjectID:    strings.TrimSpace(os.Getenv("TICKTICK_PROJECT_ID")),
	}

	if cfg.GoogleTokenType == "" {
		cfg.GoogleTokenType = "Bearer"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "./tick-sync.db"
	}
	if cfg.GooglePostSyncAction == "" {
		cfg.GooglePostSyncAction = "complete"
	}
	pollInterval, err := parsePollInterval(os.Getenv("POLL_INTERVAL"))
	if err != nil {
		return Config{}, err
	}
	cfg.PollInterval = pollInterval
	if cfg.GoogleTaskListID == "" {
		cfg.GoogleTaskListID = "@default"
	}
	if cfg.TickTickAPIBaseURL == "" {
		cfg.TickTickAPIBaseURL = "https://api.ticktick.com/open/v1"
	}
	if cfg.TickTickTimeZone == "" {
		cfg.TickTickTimeZone = "UTC"
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

func parsePollInterval(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 5 * time.Minute, nil
	}

	interval, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("POLL_INTERVAL must be a valid duration, for example 5m: %w", err)
	}
	if interval <= 0 {
		return 0, errors.New("POLL_INTERVAL must be greater than zero")
	}

	return interval, nil
}
