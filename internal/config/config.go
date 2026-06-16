// Package config loads and validates application configuration.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	googletasksync "github.com/prepin/tick-sync/internal/usecase/googletasksync"
)

// Config holds environment-driven configuration for the service.
type Config struct {
	DBPath               string                        `env:"DB_PATH" envDefault:"./tick-sync.db"`
	GooglePostSyncAction googletasksync.PostSyncAction `env:"GOOGLE_POST_SYNC_ACTION" envDefault:"complete"`
	PollInterval         time.Duration                 `env:"POLL_INTERVAL" envDefault:"5m"`

	GoogleClientID     string    `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string    `env:"GOOGLE_CLIENT_SECRET"`
	GoogleRefreshToken string    `env:"GOOGLE_REFRESH_TOKEN"`
	GoogleTokenType    string    `env:"GOOGLE_TOKEN_TYPE" envDefault:"Bearer"`
	GoogleTokenExpiry  time.Time `env:"GOOGLE_TOKEN_EXPIRY"`
	GoogleAPIEndpoint  string    `env:"GOOGLE_API_ENDPOINT"`
	GoogleTaskListID   string    `env:"GOOGLE_TASKLIST_ID" envDefault:"@default"`

	TickTickAccessToken string `env:"TICKTICK_ACCESS_TOKEN"`
	TickTickAPIBaseURL  string `env:"TICKTICK_API_BASE_URL" envDefault:"https://api.ticktick.com/open/v1"`
	TickTickTimeZone    string `env:"TICKTICK_TIME_ZONE" envDefault:"UTC"`
	TickTickProjectID   string `env:"TICKTICK_PROJECT_ID"`
}

// Load reads configuration from environment variables and applies defaults.
func Load() (Config, error) {
	_ = godotenv.Load()

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}

	switch cfg.GooglePostSyncAction {
	case googletasksync.PostSyncActionComplete:
		cfg.GooglePostSyncAction = googletasksync.PostSyncActionComplete
	case googletasksync.PostSyncActionDelete:
		cfg.GooglePostSyncAction = googletasksync.PostSyncActionDelete
	default:
		return Config{}, fmt.Errorf("unsupported GOOGLE_POST_SYNC_ACTION %q; expected complete or delete", cfg.GooglePostSyncAction)
	}

	if cfg.GoogleTokenExpiry.IsZero() {
		cfg.GoogleTokenExpiry = time.Now().Add(-time.Hour)
	}
	if cfg.PollInterval <= 0 {
		return Config{}, fmt.Errorf("POLL_INTERVAL must be greater than zero")
	}

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
