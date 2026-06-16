# Tick Sync Implementation Plan

## Goal

Implement a small Go service that synchronizes uncompleted Google Tasks into the TickTick inbox.

The first iteration is intentionally one-way:

- Source: Google Tasks
- Destination: TickTick inbox
- Polling interval: default `5m`
- Accounts: one Google Tasks account and one TickTick account
- OAuth flow: not implemented in the app for the first iteration
- Token setup: documented manually under `docs/`

## Functional Behavior

- Poll Google Tasks on startup and then every configured interval.
- Read only uncompleted Google Tasks.
- For each unprocessed Google task, create a corresponding task in the TickTick inbox.
- Preserve as many fields as practical.
- Required field mapping:
  - Google task `title` -> TickTick task title
  - Google task `notes` -> TickTick task details/content
  - Google task `due` -> TickTick task due date
- Store processed task mappings in a small SQLite database for idempotency and deduplication.
- After a task is successfully created in TickTick and recorded in the local DB, mark the Google task as completed by default.
- Support deleting the Google task instead through configuration.
- Continue processing remaining tasks when one task fails.
- Return/log a per-run summary with counts and errors.

## Configuration

Configuration is loaded from environment variables and an optional `.env` file.

Required or defaulted values:

```env
POLL_INTERVAL=5m
DB_PATH=./tick-sync.db

GOOGLE_TOKEN_FILE=./secrets/google-token.json
GOOGLE_TASKLIST_ID=@default
GOOGLE_POST_SYNC_ACTION=complete

TICKTICK_TOKEN_FILE=./secrets/ticktick-token.json
```

`GOOGLE_POST_SYNC_ACTION` values:

- `complete`, default and recommended
- `delete`

## Existing Project Layout

Use the existing directories:

```text
cmd/app/
internal/app/
internal/config/
internal/entrypoints/
internal/infra/
  clients/
  sqlite/
internal/usecase/
```

Add `docs/` for token setup documentation if it does not already exist.

## Dependencies

Planned dependencies:

- `go.uber.org/mock` for mocks and TDD around business logic
- `github.com/joho/godotenv` for `.env` loading
- `modernc.org/sqlite` for SQLite without CGO
- `golang.org/x/oauth2` for token-based HTTP clients
- `google.golang.org/api/tasks/v1` for Google Tasks API access

## TDD Approach

Start with the sync use case and mocked dependencies before implementing live clients.

Use `go.uber.org/mock/gomock` for generated mocks around small interfaces.

Initial use-case interfaces:

```go
type GoogleTasksClient interface {
    ListUncompleted(ctx context.Context) ([]GoogleTask, error)
    Complete(ctx context.Context, taskID string) error
    Delete(ctx context.Context, taskID string) error
}

type TickTickClient interface {
    CreateInboxTask(ctx context.Context, task TickTickTaskInput) (TickTickTask, error)
}

type SyncStore interface {
    IsProcessed(ctx context.Context, googleTaskID string) (bool, error)
    MarkProcessed(ctx context.Context, record SyncedTaskRecord) error
}
```

Core test cases:

- Creates a TickTick inbox task for a new uncompleted Google task.
- Preserves title, notes/details, and due date.
- Records the processed mapping after TickTick creation succeeds.
- Completes the Google task after successful processing by default.
- Deletes the Google task when `GOOGLE_POST_SYNC_ACTION=delete`.
- Skips tasks already processed in the local DB.
- Does not complete or delete the Google task if TickTick creation fails.
- Does not create duplicate TickTick tasks on later runs when the DB mapping already exists.
- Continues processing other tasks when one task fails.
- Handles an empty Google task list.
- Returns/logs a useful sync summary with created, skipped, failed, completed, and deleted counts.

## Sync Flow

Per polling cycle:

1. List uncompleted Google Tasks.
2. For each task:
   - Check whether the Google task ID already exists in the SQLite store.
   - Skip already processed tasks.
   - Create a TickTick inbox task.
   - Store the Google-to-TickTick mapping in SQLite.
   - Complete or delete the Google task based on `GOOGLE_POST_SYNC_ACTION`.
3. Continue on per-task errors.
4. Log or return a run summary.

Important failure behavior:

- If TickTick creation fails, do not write the DB mapping and do not complete/delete the Google task.
- If DB recording fails after TickTick creation, do not complete/delete the Google task. This can create a duplicate risk on the next run, so the error must be logged clearly.
- If DB recording succeeds but Google completion/deletion fails, keep the DB mapping. The next run must not create a duplicate TickTick task.

## SQLite Store

Suggested schema:

```sql
CREATE TABLE IF NOT EXISTS synced_tasks (
  google_task_id TEXT PRIMARY KEY,
  google_updated TEXT,
  google_title TEXT,
  ticktick_task_id TEXT NOT NULL,
  post_sync_action TEXT NOT NULL,
  synced_at TEXT NOT NULL
);
```

The first iteration only needs idempotent create-once behavior. Updating existing TickTick tasks when a Google task changes can be added later if needed.

## Google Client

Location: `internal/clients/google`

Responsibilities:

- Load a token from `GOOGLE_TOKEN_FILE`.
- Build an authenticated Google Tasks API client.
- List uncompleted tasks from `GOOGLE_TASKLIST_ID`.
- Mark a task completed.
- Delete a task when configured.

The service must not start an OAuth browser or terminal authorization flow in the first iteration.

## TickTick Client

Location: `internal/clients/ticktick`

Responsibilities:

- Load a token from `TICKTICK_TOKEN_FILE`.
- Create tasks in the TickTick inbox, not a specific project.
- Map Google fields into the TickTick task creation request.
- Keep TickTick HTTP request/response structs isolated from the use-case layer.

## Config Package

Location: `internal/config`

Responsibilities:

- Load `.env` when present.
- Read environment variables.
- Apply defaults.
- Validate required settings.
- Parse `POLL_INTERVAL`.
- Validate `GOOGLE_POST_SYNC_ACTION`.

## App Entry Point

Location: `cmd/app`

Responsibilities:

- Load config.
- Initialize SQLite store.
- Initialize Google and TickTick clients.
- Run one sync immediately.
- Start the polling loop.
- Handle graceful shutdown on process signals.

## Documentation

Create `docs/tokens.md` with:

- Google Cloud setup steps.
- Required Google OAuth scope: `https://www.googleapis.com/auth/tasks`.
- Instructions for obtaining a Google OAuth token JSON outside the app.
- Instructions for obtaining a TickTick token outside the app.
- Expected token file locations and shapes.
- `.env` example.
- Security warning not to commit `.env`, token files, or secrets.

## Live Testing Plan

Live service testing happens only after the core logic is covered by unit tests.

Suggested live test sequence:

1. Use a dedicated Google task list and TickTick test inbox/account.
2. Create a Google task with title, notes, and due date.
3. Run the service once.
4. Verify the task appears in the TickTick inbox.
5. Verify the Google task is completed.
6. Run the service again.
7. Verify no duplicate TickTick task is created.
8. Test a forced TickTick failure and verify the Google task remains uncompleted.
9. Test a forced Google completion failure and verify no duplicate is created on the next run.

## Future Enhancements

- Update existing TickTick tasks when Google task fields change.
- Sync completed state instead of completing/deleting source tasks after creation.
- Support multiple Google task lists.
- Support multiple TickTick accounts or projects.
- Add metrics and structured logging.
- Add a dry-run mode.
