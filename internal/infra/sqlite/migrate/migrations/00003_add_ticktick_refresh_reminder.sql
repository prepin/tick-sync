-- +goose Up
-- +goose StatementBegin
ALTER TABLE ticktick_tokens ADD COLUMN refresh_reminder_task_id TEXT;
ALTER TABLE ticktick_tokens ADD COLUMN refresh_reminder_created_at TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE ticktick_tokens DROP COLUMN refresh_reminder_created_at;
ALTER TABLE ticktick_tokens DROP COLUMN refresh_reminder_task_id;
-- +goose StatementEnd
