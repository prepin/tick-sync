-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ticktick_tokens (
  provider TEXT PRIMARY KEY,
  access_token TEXT NOT NULL,
  token_type TEXT NOT NULL,
  scope TEXT,
  expires_at TEXT,
  refresh_token TEXT,
  updated_at TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS ticktick_tokens;
