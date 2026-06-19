-- +goose Up
-- +goose StatementBegin
ALTER TABLE synced_google_tasks ADD COLUMN google_finalized_at TEXT;

UPDATE synced_google_tasks
SET google_finalized_at = synced_at
WHERE google_finalized_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE synced_google_tasks DROP COLUMN google_finalized_at;
-- +goose StatementEnd
