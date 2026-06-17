-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS oauth_tokens (
  provider TEXT PRIMARY KEY,
  access_token TEXT NOT NULL,
  token_type TEXT NOT NULL,
  scope TEXT,
  expires_at TEXT,
  refresh_token TEXT,
  updated_at TEXT NOT NULL,
  refresh_reminder_task_id TEXT,
  refresh_reminder_created_at TEXT
);

INSERT OR REPLACE INTO oauth_tokens (
  provider,
  access_token,
  token_type,
  scope,
  expires_at,
  refresh_token,
  updated_at,
  refresh_reminder_task_id,
  refresh_reminder_created_at
)
SELECT
  provider,
  access_token,
  token_type,
  scope,
  expires_at,
  refresh_token,
  updated_at,
  refresh_reminder_task_id,
  refresh_reminder_created_at
FROM ticktick_tokens;

DROP TABLE ticktick_tokens;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ticktick_tokens (
  provider TEXT PRIMARY KEY,
  access_token TEXT NOT NULL,
  token_type TEXT NOT NULL,
  scope TEXT,
  expires_at TEXT,
  refresh_token TEXT,
  updated_at TEXT NOT NULL,
  refresh_reminder_task_id TEXT,
  refresh_reminder_created_at TEXT
);

INSERT OR REPLACE INTO ticktick_tokens (
  provider,
  access_token,
  token_type,
  scope,
  expires_at,
  refresh_token,
  updated_at,
  refresh_reminder_task_id,
  refresh_reminder_created_at
)
SELECT
  provider,
  access_token,
  token_type,
  scope,
  expires_at,
  refresh_token,
  updated_at,
  refresh_reminder_task_id,
  refresh_reminder_created_at
FROM oauth_tokens
WHERE provider = 'ticktick';

DROP TABLE oauth_tokens;
-- +goose StatementEnd
