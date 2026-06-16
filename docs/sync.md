# Running Sync

The long-running sync service:

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
GOOGLE_TODAY_IMPORT_DELAY=false
TZ=Europe/Warsaw

TICKTICK_ACCESS_TOKEN=your-ticktick-access-token
TICKTICK_PROJECT_ID=
```

`GOOGLE_POST_SYNC_ACTION` values:

- `complete`, default
- `delete`

When `TICKTICK_PROJECT_ID` is empty, the client omits `projectId` and attempts to create tasks in the TickTick inbox. If TickTick returns an error like `projectId required`, set `TICKTICK_PROJECT_ID` to a real project ID as a fallback.

`TZ` uses conventional IANA timezone names, such as `Europe/Warsaw` or `America/New_York`. When `TZ` is not set, the app uses the system local timezone for date decisions and omits TickTick's `timeZone` field.

When `GOOGLE_TODAY_IMPORT_DELAY=true`, Google tasks due today are left in Google Tasks and imported only after they become overdue. This avoids importing tasks created from commands like "remind me in 15 minutes" as all-day TickTick tasks, because the Google Tasks API does not expose due times.

Stop it with Ctrl+C or SIGTERM.

## Safety

`go run ./cmd/app` mutates data. It creates TickTick tasks and then completes or deletes Google tasks depending on `GOOGLE_POST_SYNC_ACTION`. It runs one sync immediately on startup, then repeats every `POLL_INTERVAL`.
