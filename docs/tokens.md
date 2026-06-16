# Google Tasks Token Setup

The first iteration does not run an OAuth browser flow and does not read separate secret files.

All Google OAuth values are provided through environment variables or a local `.env` file.

## Required Scope

Use this Google OAuth scope:

```text
https://www.googleapis.com/auth/tasks
```

## Google Cloud Setup

1. Open [Google Cloud Console](https://console.cloud.google.com/).
2. Create or select a project.
3. Enable the [Google Tasks API](https://console.cloud.google.com/apis/library/tasks.googleapis.com).
4. Configure the [OAuth consent screen](https://console.cloud.google.com/apis/credentials/consent).
5. Create [OAuth client credentials](https://console.cloud.google.com/apis/credentials).
6. If using OAuth 2.0 Playground, create a `Web application` OAuth client.
7. If Google Cloud requires an authorized JavaScript origin, add:

```text
https://developers.google.com
```

8. Add this authorized redirect URI to the Web application client:

```text
https://developers.google.com/oauthplayground
```

## Environment Variables

Create a local `.env` file or export these variables in your shell:

```env
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REFRESH_TOKEN=your-refresh-token
GOOGLE_TOKEN_TYPE=Bearer
GOOGLE_TOKEN_EXPIRY=2026-06-10T12:00:00Z
GOOGLE_TASKLIST_ID=@default
```

Required variables:

- `GOOGLE_CLIENT_ID`
- `GOOGLE_CLIENT_SECRET`
- `GOOGLE_REFRESH_TOKEN`

Optional variables:

- `GOOGLE_TOKEN_TYPE`, defaults to `Bearer`
- `GOOGLE_TOKEN_EXPIRY`, defaults to an already-expired time so the token refreshes immediately
- `GOOGLE_TASKLIST_ID`, defaults to `@default`

## Getting Tokens

Use an external OAuth tool to obtain an access token and refresh token for the Google Tasks scope.

The service expects tokens as environment variables, not as JSON files.

One practical approach is to use [OAuth 2.0 Playground](https://developers.google.com/oauthplayground/).

OAuth Playground requires a `Web application` OAuth client with this authorized redirect URI:

```text
https://developers.google.com/oauthplayground
```

A `Desktop app` OAuth client can fail here with `redirect_uri_mismatch`, because OAuth Playground redirects through its hosted URL.

If Google Cloud requires an authorized JavaScript origin for the Web application client, use:

```text
https://developers.google.com
```

Steps:

1. Open [OAuth 2.0 Playground](https://developers.google.com/oauthplayground/).
2. Click the gear icon in the top-right.
3. Enable `Use your own OAuth credentials`.
4. Keep `OAuth flow` set to `Server-side`.
5. Enter your Google OAuth client ID and client secret from Google Cloud Console.
6. In the left scope input, manually enter `https://www.googleapis.com/auth/tasks`.
7. Click `Authorize APIs`.
8. Select your Google account and allow access.
9. After redirecting back to OAuth Playground, click `Exchange authorization code for tokens`.
10. Copy the refresh token into `.env`.

Example `.env` values after token exchange:

```env
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REFRESH_TOKEN=copy-refresh-token-from-playground
GOOGLE_TOKEN_TYPE=Bearer
GOOGLE_TASKLIST_ID=@default
```

If OAuth Playground shows an expiry timestamp, set `GOOGLE_TOKEN_EXPIRY` as an RFC3339 timestamp:

```env
GOOGLE_TOKEN_EXPIRY=2026-06-10T12:00:00Z
```

If you only see `expires_in`, you can omit `GOOGLE_TOKEN_EXPIRY`. The service treats a missing expiry as already expired and refreshes immediately using `GOOGLE_REFRESH_TOKEN`.

## Missing Refresh Token

If OAuth Playground does not return a refresh token:

1. Open the OAuth Playground settings panel.
2. Make sure access type is `Offline`.
3. Revoke the existing app grant from [Google Account permissions](https://myaccount.google.com/permissions).
4. Authorize the scope again in OAuth Playground.

Google often returns a refresh token only on the first consent for a client, user, and scope combination.

## Access Denied During Authorization

If Google returns `Error 403: access_denied` during OAuth Playground authorization, check the OAuth consent screen in Google Cloud Console.

For apps in `Testing` publishing status:

1. Open the [OAuth consent screen](https://console.cloud.google.com/apis/credentials/consent).
2. Confirm the app publishing status is `Testing`.
3. Add the Google account you are authorizing with under `Test users`.
4. Save the consent screen changes.
5. Retry authorization in OAuth Playground.

If you do not want to manage test users, publish the app to production. For this personal tool, keeping the app in testing and adding yourself as a test user is usually simpler.

Also confirm that OAuth Playground is using the same OAuth client ID from the project where the Google Tasks API is enabled and the consent screen is configured.

## Invalid Client Secret During Token Exchange

If token exchange returns `invalid_client` with `The provided client secret is invalid`, OAuth Playground is using a client secret that does not match the client ID.

Check the following:

1. Open [Google Cloud credentials](https://console.cloud.google.com/apis/credentials).
2. Open the exact `Web application` OAuth client used in OAuth Playground.
3. Copy the `Client ID` from that client into OAuth Playground.
4. Copy the `Client secret` value from that same client into OAuth Playground.
5. Do not use the secret from a different OAuth client, such as an older `Desktop app` client.
6. Do not use the `Secret ID`; use the actual `Client secret` value.
7. If unsure, create a new client secret for the Web application client and use the new value.

The `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` in `.env` must be the same pair used to generate the access and refresh tokens.

## Running The Service

```sh
go run ./cmd/app
```

The command starts the long-running sync service. It syncs Google Tasks to TickTick on startup and then on every `POLL_INTERVAL`.

For TickTick token setup, see [`docs/ticktick.md`](ticktick.md). For sync details, see [`docs/sync.md`](sync.md).

## Security

Do not commit `.env`, access tokens, refresh tokens, client secrets, or copied OAuth responses.
