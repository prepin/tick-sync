# TickTick Token Setup

The first TickTick integration uses an access token from environment variables. The app does not run a TickTick OAuth flow yet.

## Required Scopes

Use these TickTick OAuth scopes:

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
http://localhost:8080/callback
```

The local callback server does not need to exist for this manual setup. After authorization, copy the `code` value from the browser address bar.

## Get Access Token

Open this authorization URL in your browser after replacing `YOUR_CLIENT_ID`:

```text
https://ticktick.com/oauth/authorize?scope=tasks:read%20tasks:write&client_id=YOUR_CLIENT_ID&state=tick-sync&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback&response_type=code
```

Approve access. TickTick redirects to a URL like:

```text
http://localhost:8080/callback?code=AUTH_CODE&state=tick-sync
```

The page may fail to load because no local server is running. That is fine. Copy `AUTH_CODE` from the address bar.

Exchange the authorization code for an access token:

```sh
curl -X POST https://ticktick.com/oauth/token \
  -u 'YOUR_CLIENT_ID:YOUR_CLIENT_SECRET' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode 'code=AUTH_CODE' \
  --data-urlencode 'grant_type=authorization_code' \
  --data-urlencode 'scope=tasks:read tasks:write' \
  --data-urlencode 'redirect_uri=http://localhost:8080/callback'
```

The response should contain:

```json
{
  "access_token": "...",
  "token_type": "bearer",
  "expires_in": 86400
}
```

Copy `access_token` into `.env` as `TICKTICK_ACCESS_TOKEN`.

TickTick's docs currently document the authorization code exchange clearly, but do not clearly document refresh-token behavior. If the token expires and no refresh token is returned, repeat this manual flow until the app has a proper OAuth/token-refresh implementation.

## Environment Variables

```env
TICKTICK_ACCESS_TOKEN=your-ticktick-access-token
TICKTICK_API_BASE_URL=https://api.ticktick.com/open/v1
TICKTICK_TIME_ZONE=UTC
TICKTICK_PROJECT_ID=
```

Required variables when running sync:

- `TICKTICK_ACCESS_TOKEN`

Optional variables:

- `TICKTICK_API_BASE_URL`, defaults to `https://api.ticktick.com/open/v1`
- `TICKTICK_TIME_ZONE`, defaults to `UTC`
- `TICKTICK_PROJECT_ID`, defaults to empty

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
yyyy-MM-dd'T'HH:mm:ssZ
```

Example:

```text
2026-06-12T00:00:00+0000
```

## Security

Do not commit `.env`, access tokens, client secrets, or copied OAuth responses.

For running the sync command, see [`docs/sync.md`](sync.md).
