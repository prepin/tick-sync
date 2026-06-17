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
	DBPath                 string                        `env:"DB_PATH" envDefault:"./tick-sync.db"`
	HTTPAddr               string                        `env:"HTTP_ADDR" envDefault:":8080"`
	GooglePostSyncAction   googletasksync.PostSyncAction `env:"GOOGLE_POST_SYNC_ACTION" envDefault:"complete"`
	GoogleTodayImportDelay bool                          `env:"GOOGLE_TODAY_IMPORT_DELAY" envDefault:"false"`
	PollInterval           time.Duration                 `env:"POLL_INTERVAL" envDefault:"5m"`
	TZ                     string                        `env:"TZ"`
	Location               *time.Location                `env:"-"`

	GoogleClientID     string    `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string    `env:"GOOGLE_CLIENT_SECRET"`
	GoogleRefreshToken string    `env:"GOOGLE_REFRESH_TOKEN"`
	GoogleTokenType    string    `env:"GOOGLE_TOKEN_TYPE" envDefault:"Bearer"`
	GoogleTokenExpiry  time.Time `env:"GOOGLE_TOKEN_EXPIRY"`
	GoogleAPIEndpoint  string    `env:"GOOGLE_API_ENDPOINT"`
	GoogleTaskListID   string    `env:"GOOGLE_TASKLIST_ID" envDefault:"@default"`

	TickTickClientID     string `env:"TICKTICK_CLIENT_ID"`
	TickTickClientSecret string `env:"TICKTICK_CLIENT_SECRET"`
	TickTickRedirectURL  string `env:"TICKTICK_REDIRECT_URL" envDefault:"http://localhost:8080/ticktick/callback"`
	TickTickAuthURL      string `env:"TICKTICK_AUTH_URL" envDefault:"https://ticktick.com/oauth/authorize"`
	TickTickTokenURL     string `env:"TICKTICK_TOKEN_URL" envDefault:"https://ticktick.com/oauth/token"`
	TickTickAPIBaseURL   string `env:"TICKTICK_API_BASE_URL" envDefault:"https://api.ticktick.com/open/v1"`
	TickTickProjectID    string `env:"TICKTICK_PROJECT_ID"`
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
	if cfg.TZ == "" {
		cfg.Location = time.Local
	} else {
		location, err := time.LoadLocation(cfg.TZ)
		if err != nil {
			return Config{}, fmt.Errorf("load TZ %q: %w", cfg.TZ, err)
		}
		cfg.Location = location
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
