-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS synced_google_tasks (
  google_task_id TEXT PRIMARY KEY,
  google_updated TEXT,
  google_title TEXT,
  ticktick_task_id TEXT NOT NULL,
  post_sync_action TEXT NOT NULL,
  synced_at TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS synced_google_tasks;
