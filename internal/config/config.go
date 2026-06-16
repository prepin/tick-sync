// Package config loads and validates application configuration.
package config

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

// Config holds environment-driven configuration for the service.
type Config struct {
	DBPath               string
	GooglePostSyncAction googletasksync.PostSyncAction
	PollInterval         time.Duration

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRefreshToken string
	GoogleTokenType    string
	GoogleTokenExpiry  time.Time
	GoogleAPIEndpoint  string
	GoogleTaskListID   string

	TickTickAccessToken string
	TickTickAPIBaseURL  string
	TickTickTimeZone    string
	TickTickProjectID   string
}

// Load reads configuration from environment variables and applies defaults.
func Load() (Config, error) {
	_ = godotenv.Load()

	rawAction := cmp.Or(env("GOOGLE_POST_SYNC_ACTION"), "complete")

	var postSyncAction googletasksync.PostSyncAction
	switch rawAction {
	case string(googletasksync.PostSyncActionComplete):
		postSyncAction = googletasksync.PostSyncActionComplete
	case string(googletasksync.PostSyncActionDelete):
		postSyncAction = googletasksync.PostSyncActionDelete
	default:
		return Config{}, fmt.Errorf("unsupported GOOGLE_POST_SYNC_ACTION %q; expected complete or delete", rawAction)
	}

	cfg := Config{
		DBPath:               cmp.Or(env("DB_PATH"), "./tick-sync.db"),
		GooglePostSyncAction: postSyncAction,
		GoogleClientID:       env("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:   env("GOOGLE_CLIENT_SECRET"),
		GoogleRefreshToken:   env("GOOGLE_REFRESH_TOKEN"),
		GoogleTokenType:      cmp.Or(env("GOOGLE_TOKEN_TYPE"), "Bearer"),
		GoogleTaskListID:     cmp.Or(env("GOOGLE_TASKLIST_ID"), "@default"),
		TickTickAccessToken:  env("TICKTICK_ACCESS_TOKEN"),
		TickTickAPIBaseURL:   cmp.Or(env("TICKTICK_API_BASE_URL"), "https://api.ticktick.com/open/v1"),
		TickTickTimeZone:     cmp.Or(env("TICKTICK_TIME_ZONE"), "UTC"),
		TickTickProjectID:    env("TICKTICK_PROJECT_ID"),
	}

	pollInterval, err := parsePollInterval(env("POLL_INTERVAL"))
	if err != nil {
		return Config{}, err
	}
	cfg.PollInterval = pollInterval

	expiry, err := parseTokenExpiry(env("GOOGLE_TOKEN_EXPIRY"))
	if err != nil {
		return Config{}, err
	}
	cfg.GoogleTokenExpiry = expiry

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate returns an error if required configuration values are missing.
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

func env(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func parseTokenExpiry(value string) (time.Time, error) {
	if value == "" {
		return time.Now().Add(-time.Hour), nil
	}

	expiry, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, errors.New(
			"GOOGLE_TOKEN_EXPIRY must be an RFC3339 timestamp, for example 2026-06-10T12:00:00Z",
		)
	}

	return expiry, nil
}

func parsePollInterval(value string) (time.Duration, error) {
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
