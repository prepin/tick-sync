# Google Tasks Token Setup

The app runs a local browser-based Google OAuth flow and stores tokens in SQLite. `GOOGLE_REFRESH_TOKEN` in `.env` is ignored.

## Required Scope

The app requests this Google OAuth scope:

```text
https://www.googleapis.com/auth/tasks
```

## Google Cloud Setup

1. Open [Google Cloud Console](https://console.cloud.google.com/).
2. Create or select a project.
3. Enable the [Google Tasks API](https://console.cloud.google.com/apis/library/tasks.googleapis.com).
4. Configure the [OAuth consent screen](https://console.cloud.google.com/apis/credentials/consent).
5. Create [OAuth client credentials](https://console.cloud.google.com/apis/credentials) for a `Web application`.
6. Add this authorized redirect URI:

```text
http://localhost:8080/google/callback
```

## Environment Variables

```env
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/google/callback
GOOGLE_TASKLIST_ID=@default
```

Required variables:

- `GOOGLE_CLIENT_ID`
- `GOOGLE_CLIENT_SECRET`

Optional variables:

- `GOOGLE_REDIRECT_URL`, defaults to `http://localhost:8080/google/callback`
- `GOOGLE_TASKLIST_ID`, defaults to `@default`

## Connect Google Tasks

Start the service:

```sh
go run ./cmd/app
```

Open the local start page:

```text
http://localhost:8080/
```

Click `Connect Google Tasks`, approve access, and the callback page saves the token in the configured SQLite database.

The auth request uses `access_type=offline` and `prompt=consent` so Google returns a refresh token. If Google still does not return one, revoke the app grant in [Google Account permissions](https://myaccount.google.com/permissions) and connect again.

## Invalid Grant

If sync logs `invalid_grant`, the stored Google refresh token was revoked or expired. Open `http://localhost:8080/` and connect Google Tasks again.

For TickTick token setup, see [`docs/ticktick.md`](ticktick.md). For sync details, see [`docs/sync.md`](sync.md).

## Security

Do not commit `.env`, tokens, client secrets, or copied OAuth responses. OAuth tokens are stored in the configured SQLite database.
