# Tick Sync

## What it does

Tick Sync is a small service that copies your tasks from Google Tasks into TickTick. Since you can't set up on-device task import on Android as you can on iOS, this is a convenient workaround.

Tick Sync:

- Polls Google Tasks on a configurable interval.
- Skips tasks that were already synced.
- Creates matching tasks in TickTick.
- Completes Google Tasks after successful sync by default. You can configure it to delete original tasks instead.

## Caveats

- The Google Tasks API exposes task due dates, but not due times. Tick Sync does not sync task times yet.
- TickTick does not allow automatic token renewal. You will need to reconnect TickTick manually about every six months, but Tick Sync creates a reminder task for you before the token expires.

## Installation

Create a `.env` file from `.env.example`, then fill in your Google and TickTick OAuth credentials.

Required setup:

- Google OAuth client with the Google Tasks API enabled.
- TickTick OAuth app.
- A persistent `DB_PATH` for the SQLite database.

Common environment variables:

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `GOOGLE_CLIENT_ID` | Yes | | Google OAuth client ID. |
| `GOOGLE_CLIENT_SECRET` | Yes | | Google OAuth client secret. |
| `TICKTICK_CLIENT_ID` | Yes | | TickTick OAuth client ID. |
| `TICKTICK_CLIENT_SECRET` | Yes | | TickTick OAuth client secret. |
| `DB_PATH` | No | `./tick-sync.db` | SQLite database path. |
| `HTTP_ADDR` | No | `:8080` | Local HTTP server address. |
| `HTTP_BASIC_AUTH_USERNAME` | No | `tick-sync` | Username for HTTP Basic Auth when `HTTP_BASIC_AUTH_PASSWORD` is set. |
| `HTTP_BASIC_AUTH_PASSWORD` | No | | Password for HTTP Basic Auth. Empty disables HTTP Basic Auth. Set this when exposing the HTTP server beyond localhost. |
| `HTTP_CLIENT_TIMEOUT` | No | `30s` | Timeout for outbound OAuth, Google Tasks, and TickTick HTTP requests. |
| `POLL_INTERVAL` | No | `5m` | How often to sync Google Tasks to TickTick. |
| `GOOGLE_POST_SYNC_ACTION` | No | `complete` | `complete` or `delete` Google Tasks after successful sync. |
| `GOOGLE_TODAY_IMPORT_DELAY` | No | `false` | Delay importing tasks due today until they become overdue. |
| `GOOGLE_TASKLIST_ID` | No | `@default` | Google task list to sync from. |
| `TICKTICK_PROJECT_ID` | No | | TickTick project ID. Empty attempts to create tasks in the inbox. |
| `TZ` | No | system local timezone | IANA timezone for date handling, for example `Europe/Warsaw`. |

### With Docker

Set the database path inside the mounted data directory:

```env
DB_PATH=/data/tick-sync.db
HTTP_ADDR=:8080
```

Run with Docker:

```sh
docker run --rm \
  --env-file .env \
  -p 8080:8080 \
  -v "$PWD/data:/data" \
  ghcr.io/prepin/tick-sync:latest
```

Or use Docker Compose:

```yaml
services:
  tick-sync:
    image: ghcr.io/prepin/tick-sync:latest
    env_file: .env
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    restart: unless-stopped
```

### With Go install

If you already have Go installed, install the latest version from source:

```sh
go install github.com/prepin/tick-sync/cmd/app@latest
```

Then create and fill your `.env` file before running the installed `app` binary.

### Manually

Download and extract the latest binary for your platform from the [GitHub releases page](https://github.com/prepin/tick-sync/releases/latest), then create and fill your `.env` file.

```sh
$EDITOR .env
./tick-sync
```

## Connecting TickTick and Google Tasks

1. Register OAuth clients for Google Tasks and TickTick. Use the [setup guide](docs/setup.md) for the detailed checklist.
2. Configure the Google redirect URL as `http://localhost:8080/google/callback`.
3. Configure the TickTick redirect URL as `http://localhost:8080/ticktick/callback`.
4. Put the client IDs and client secrets in `.env`.
5. Start the service, then open:

```text
http://localhost:8080/
```

6. Click `Connect Google Tasks`, approve access, then click `Connect TickTick` and approve access. Tokens are stored in SQLite.

If `HTTP_ADDR` is reachable from other machines, set `HTTP_BASIC_AUTH_PASSWORD` so the setup UI and OAuth callback endpoints require HTTP Basic Auth.

The unauthenticated `GET /healthz` endpoint returns `204 No Content` for container or uptime health checks.

More details:

- [Setup guide](docs/setup.md)
- [Sync behavior](docs/sync.md)

## License

MIT License. See [LICENSE](LICENSE).
