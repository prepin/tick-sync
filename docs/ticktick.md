# TickTick Token Setup

The app runs a local browser-based TickTick OAuth flow and stores the returned access token in SQLite. You no longer need to copy `TICKTICK_ACCESS_TOKEN` into `.env`.

## Required Scopes

The app requests these TickTick OAuth scopes:

```text
tasks:read tasks:write
```

## Create TickTick App

1. Open [TickTick Developer Center](https://developer.ticktick.com/manage).
2. Sign in.
3. Create an app.
4. Copy the client ID and client secret.
5. Set the OAuth redirect URL to:

```text
http://localhost:8080/ticktick/callback
```

## Environment Variables

```env
HTTP_ADDR=:8080
TICKTICK_REMINDER_INTERVAL=24h
TICKTICK_CLIENT_ID=your-ticktick-client-id
TICKTICK_CLIENT_SECRET=your-ticktick-client-secret
TICKTICK_REDIRECT_URL=http://localhost:8080/ticktick/callback
TICKTICK_API_BASE_URL=https://api.ticktick.com/open/v1
TICKTICK_PROJECT_ID=
```

Optional variables:

- `HTTP_ADDR`, defaults to `:8080`
- `TICKTICK_REMINDER_INTERVAL`, defaults to `24h`
- `TICKTICK_REDIRECT_URL`, defaults to `http://localhost:8080/ticktick/callback`
- `TICKTICK_API_BASE_URL`, defaults to `https://api.ticktick.com/open/v1`
- `TICKTICK_PROJECT_ID`, defaults to empty
- `TZ`, defaults to the system local timezone when unset

## Connect TickTick

Start the service:

```sh
go run ./cmd/app
```

Open the local start page:

```text
http://localhost:8080/
```

Click `Connect TickTick`, approve access, and the callback page saves the token in the configured SQLite database.

The sync job starts before TickTick is connected. If Google tasks need to be copied while no TickTick token exists, that sync tick logs a missing token error and the next poll retries.

If the stored TickTick token expires in less than two weeks, the service creates one medium-priority TickTick task named `Refresh TickTick token`. The reminder is created once per access token and has no due date.

## Inbox Behavior

When `TICKTICK_PROJECT_ID` is empty, the client omits `projectId` from `POST /open/v1/task`. This attempts to create the task in the TickTick inbox.

TickTick's official Open API docs mark `projectId` as required for task creation, but inbox creation may work by omitting it. If live sync returns an error like `projectId required`, set `TICKTICK_PROJECT_ID` to a real TickTick project ID as a fallback.

## Open API Docs

Official docs:

```text
https://developer.ticktick.com/docs/openapi.md
```

Task creation endpoint:

```http
POST https://api.ticktick.com/open/v1/task
Authorization: Bearer <access_token>
Content-Type: application/json
```

Due dates use this format:

```text
```

Example:

```text
2026-06-12T00:00:00+0000
```

## Security

Do not commit `.env`, client secrets, access tokens, or copied OAuth responses. The access token is stored in the configured SQLite database.

For running the sync command, see [`docs/sync.md`](sync.md).
