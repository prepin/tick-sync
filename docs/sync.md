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
- Retries Google completion or deletion on later runs if TickTick creation was recorded but Google finalization failed.
- Continues processing other tasks when one task fails.

## Environment Variables

```env
DB_PATH=./tick-sync.db
HTTP_ADDR=:8080
HTTP_BASIC_AUTH_USERNAME=tick-sync
HTTP_BASIC_AUTH_PASSWORD=
HTTP_CLIENT_TIMEOUT=30s
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

`GET /healthz` returns `204 No Content` and is not protected by HTTP Basic Auth so container and uptime checks can call it without credentials.

`HTTP_CLIENT_TIMEOUT` limits outbound OAuth, Google Tasks, and TickTick API requests. It defaults to `30s` so stalled external calls do not block sync jobs indefinitely.

If the stored TickTick token expires in less than two weeks, the service creates one medium-priority TickTick task reminding you to refresh it. This reminder is created once per token and has no due date.

If the stored TickTick token expires, the service creates a separate reminder in the configured Google task list. The earlier TickTick reminder does not suppress this Google reminder.

The Google reminder uses the next noon in the `TZ` timezone, or the system timezone if `TZ` is unset:

- Before or exactly 12 PM (noon), the due date is today.
- After noon, the due date is tomorrow.

The Google Tasks API stores only the date and discards the time. The reminder notes include the intended noon time and timezone. This does not schedule a noon notification.

The service records the Google reminder separately for each TickTick token. Repeated checks and restarts do not create another reminder after that record succeeds. A new TickTick access token resets the record. A crash between task creation and database recording can cause a duplicate reminder.

The Google reminder contains the reserved notes line `[tick-sync:ticktick-token-expired]`. Sync leaves tasks with this exact line in Google Tasks, without copying, completing, or deleting them. These reminders remain in Google Tasks after reconnection.

The Google reminder requires usable Google credentials. If creation fails, the service logs the error and retries at the next check. The expiry check uses the stored TickTick expiry time. It does not detect revoked tokens before that time or tokens without a known expiry.

`TICKTICK_REMINDER_INTERVAL` controls both reminder checks. The service checks at startup, then repeats at this interval. The default is `24h`, so expiry detection can take up to 24 hours.

Stop it with Ctrl+C or SIGTERM.

## Safety

`go run ./cmd/app` mutates data. It creates TickTick tasks and then completes or deletes Google tasks depending on `GOOGLE_POST_SYNC_ACTION`. It runs one sync immediately on startup, then repeats every `POLL_INTERVAL`.
