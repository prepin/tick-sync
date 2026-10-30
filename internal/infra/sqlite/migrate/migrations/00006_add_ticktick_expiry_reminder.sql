-- +goose Up
ALTER TABLE oauth_tokens ADD COLUMN expiry_reminder_task_id TEXT;
ALTER TABLE oauth_tokens ADD COLUMN expiry_reminder_created_at TEXT;

-- +goose Down
ALTER TABLE oauth_tokens DROP COLUMN expiry_reminder_created_at;
ALTER TABLE oauth_tokens DROP COLUMN expiry_reminder_task_id;
