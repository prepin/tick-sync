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
HTTP_ADDR=:8080
HTTP_BASIC_AUTH_USERNAME=tick-sync
HTTP_BASIC_AUTH_PASSWORD=
POLL_INTERVAL=5m
TICKTICK_REMINDER_INTERVAL=24h
GOOGLE_POST_SYNC_ACTION=complete
GOOGLE_TODAY_IMPORT_DELAY=false
TZ=Europe/Warsaw

TICKTICK_CLIENT_ID=your-ticktick-client-id
TICKTICK_CLIENT_SECRET=your-ticktick-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/google/callback
TICKTICK_REDIRECT_URL=http://localhost:8080/ticktick/callback
TICKTICK_PROJECT_ID=
```

`GOOGLE_POST_SYNC_ACTION` values:

- `complete`, default
- `delete`

When `TICKTICK_PROJECT_ID` is empty, the client omits `projectId` and attempts to create tasks in the TickTick inbox. If TickTick returns an error like `projectId required`, set `TICKTICK_PROJECT_ID` to a real project ID as a fallback.

`TZ` uses conventional IANA timezone names, such as `Europe/Warsaw` or `America/New_York`. When `TZ` is not set, the app uses the system local timezone for date decisions and omits TickTick's `timeZone` field.

When `GOOGLE_TODAY_IMPORT_DELAY=true`, Google tasks due today are left in Google Tasks and imported only after they become overdue. This avoids importing tasks created from commands like "remind me in 15 minutes" as all-day TickTick tasks, because the Google Tasks API does not expose due times.

The service starts even before Google Tasks or TickTick are connected. Complete both auth flows at `http://localhost:8080/`. Until then, sync attempts report the missing provider token and the next poll retries.

When `HTTP_BASIC_AUTH_PASSWORD` is set, the setup UI and OAuth callback endpoints require HTTP Basic Auth. Set it when `HTTP_ADDR` is reachable from other machines.

If the stored TickTick token expires in less than two weeks, the service creates one medium-priority TickTick task reminding you to refresh it. This reminder is created once per token and has no due date.

`TICKTICK_REMINDER_INTERVAL` controls how often the token reminder check runs. It defaults to `24h`.

Stop it with Ctrl+C or SIGTERM.

## Safety

`go run ./cmd/app` mutates data. It creates TickTick tasks and then completes or deletes Google tasks depending on `GOOGLE_POST_SYNC_ACTION`. It runs one sync immediately on startup, then repeats every `POLL_INTERVAL`.
