package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/prepin/tick-sync/internal/consts"
)

type Config struct {
	DBPath               string
	GooglePostSyncAction consts.PostSyncAction
	PollInterval         time.Duration

	GoogleClientID     string
	GoogleClientSecret string
	GoogleAccessToken  string
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

func Load() (Config, error) {
	_ = godotenv.Load()

	rawAction := env("GOOGLE_POST_SYNC_ACTION")
	if rawAction == "" {
		rawAction = "complete"
	}

	var postSyncAction consts.PostSyncAction
	switch rawAction {
	case string(consts.PostSyncActionComplete):
		postSyncAction = consts.PostSyncActionComplete
	case string(consts.PostSyncActionDelete):
		postSyncAction = consts.PostSyncActionDelete
	default:
		return Config{}, fmt.Errorf("unsupported GOOGLE_POST_SYNC_ACTION %q; expected complete or delete", rawAction)
	}

	cfg := Config{
		DBPath:               env("DB_PATH"),
		GooglePostSyncAction: postSyncAction,
		GoogleClientID:       env("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:   env("GOOGLE_CLIENT_SECRET"),
		GoogleAccessToken:    env("GOOGLE_ACCESS_TOKEN"),
		GoogleRefreshToken:   env("GOOGLE_REFRESH_TOKEN"),
		GoogleTokenType:      env("GOOGLE_TOKEN_TYPE"),
		GoogleTaskListID:     env("GOOGLE_TASKLIST_ID"),
		TickTickAccessToken:  env("TICKTICK_ACCESS_TOKEN"),
		TickTickAPIBaseURL:   env("TICKTICK_API_BASE_URL"),
		TickTickTimeZone:     env("TICKTICK_TIME_ZONE"),
		TickTickProjectID:    env("TICKTICK_PROJECT_ID"),
	}

	if cfg.GoogleTokenType == "" {
		cfg.GoogleTokenType = "Bearer"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "./tick-sync.db"
	}
	pollInterval, err := parsePollInterval(env("POLL_INTERVAL"))
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
		return time.Time{}, errors.New("GOOGLE_TOKEN_EXPIRY must be an RFC3339 timestamp, for example 2026-06-10T12:00:00Z")
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
