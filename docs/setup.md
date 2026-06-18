# Setup

Use this checklist to create the OAuth clients Tick Sync needs. After registration, put the client IDs and client secrets in your `.env` file.

## Google Tasks

1. Open [Google Cloud Console](https://console.cloud.google.com/).
2. Create or select a project.
3. Enable the [Google Tasks API](https://console.cloud.google.com/apis/library/tasks.googleapis.com).
4. Configure the [OAuth consent screen](https://console.cloud.google.com/apis/credentials/consent).
5. Create OAuth client credentials for a `Web application`.
6. Add this authorized redirect URI:

```text
http://localhost:8080/google/callback
```

7. Copy the client ID and client secret into `.env`:

```env
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
```

Tick Sync requests this Google OAuth scope:

```text
https://www.googleapis.com/auth/tasks
```

The auth request uses offline access so Google returns a refresh token. If sync logs `invalid_grant`, the stored Google refresh token was revoked or expired. Open `http://localhost:8080/` and connect Google Tasks again.

## TickTick

1. Open [TickTick Developer Center](https://developer.ticktick.com/manage).
2. Sign in.
3. Create an app.
4. Add this OAuth redirect URL:

```text
http://localhost:8080/ticktick/callback
```

5. Copy the client ID and client secret into `.env`:

```env
TICKTICK_CLIENT_ID=your-ticktick-client-id
TICKTICK_CLIENT_SECRET=your-ticktick-client-secret
```

Tick Sync requests these TickTick OAuth scopes:

```text
tasks:read tasks:write
```

TickTick does not allow automatic token renewal. If the stored TickTick token expires in less than two weeks, Tick Sync creates one medium-priority TickTick task named `Refresh TickTick token`. The reminder is created once per access token and has no due date.

When `TICKTICK_PROJECT_ID` is empty, Tick Sync attempts to create tasks in the TickTick inbox. If TickTick returns an error like `projectId required`, set `TICKTICK_PROJECT_ID` to a real TickTick project ID.

## Connect Accounts

Start Tick Sync and open:

```text
http://localhost:8080/
```

Click `Connect Google Tasks`, approve access, then click `Connect TickTick` and approve access. OAuth tokens are stored in the configured SQLite database.

## Security

Do not commit `.env`, client secrets, access tokens, or copied OAuth responses.
