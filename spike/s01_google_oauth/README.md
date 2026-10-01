# S01: Google OAuth and Drive spike

This throwaway spike verifies the Google OAuth flow using PKCE and an initial Drive API call.

## Prereqs

- `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` must be exported in the shell or loaded from the repo `.env` file.
- The redirect URI configured in Google Cloud Console must match the local base URL used here.

## Run

```bash
set -a
source ../../.env
set +a
go run ./spike/s01_google_oauth
```

Then open:

```text
http://localhost:3030/login
```

Expected behavior:

- Google redirects to the consent screen
- a refresh token and access token are exchanged successfully
- the app lists a few Drive files on the callback page

This is intentionally throwaway code used to answer Phase 1 blockers before implementing the real provider.
