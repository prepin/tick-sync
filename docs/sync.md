# Running Sync

The CLI has two modes.

Read-only Google task listing:

```sh
go run ./cmd/cli
```

One sync run from Google Tasks to TickTick:

```sh
go run ./cmd/cli sync
```

Long-running sync service:

```sh
go run ./cmd/app
```

## Sync Behavior

The sync command:

- Lists uncompleted Google Tasks.
- Skips Google tasks already recorded in SQLite.
- Creates matching tasks in TickTick.
- Records the Google-to-TickTick mapping in `synced_google_tasks`.
- Completes the Google task by default after successful TickTick creation and DB recording.
- Continues processing other tasks when one task fails.

## Environment Variables

```env
DB_PATH=./tick-sync.db
POLL_INTERVAL=5m
GOOGLE_POST_SYNC_ACTION=complete

TICKTICK_ACCESS_TOKEN=your-ticktick-access-token
TICKTICK_PROJECT_ID=
```

`GOOGLE_POST_SYNC_ACTION` values:

- `complete`, default
- `delete`

When `TICKTICK_PROJECT_ID` is empty, the client omits `projectId` and attempts to create tasks in the TickTick inbox. If TickTick returns an error like `projectId required`, set `TICKTICK_PROJECT_ID` to a real project ID as a fallback.

`cmd/app` runs one sync immediately on startup, then repeats every `POLL_INTERVAL`. Stop it with Ctrl+C or SIGTERM.

## Safety

`go run ./cmd/cli sync` mutates data. It creates TickTick tasks and then completes or deletes Google tasks depending on `GOOGLE_POST_SYNC_ACTION`.

`go run ./cmd/app` performs the same mutating sync repeatedly.
