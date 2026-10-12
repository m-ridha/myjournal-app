# S01: Google OAuth and Drive spike

This throwaway spike verifies the Google OAuth flow using PKCE and an initial Drive API call.

## Prereqs

- `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` must be exported in the shell or loaded from the repo `.env` file.
- The redirect URI configured in Google Cloud Console must match the local base URL used here.

## Run

```bash
set -a
source ./.env
set +a
go run ./spike/s01_google_oauth
```

For a fresh grant, open this once after removing the app from the test account's third-party access list:

```text
http://localhost:3030/login?force_consent=1
```

After approving the grant, the callback reports only boolean diagnostics and the number of visible Drive files. It never prints token or ID-token values. Record:

- `refresh_token_present`
- `refresh_renewal_succeeded`
- `refresh_drive_list_succeeded`
- `id_token_present`, `id_token_valid`, `subject_claim_present`, and `email_claim_present`
- `drive_list_succeeded` and `drive_files_visible`

The spike simulates an expired access token using a copy of the exchanged token, refreshes it, and makes a second Drive list call. For the subsequent-consent comparison, open `/login` without `force_consent=1` and record whether `refresh_token_present` is true. Google may omit a refresh token on a subsequent grant.

Do not paste the callback URL, authorization code, state, or any token into notes or chat. A zero file count still indicates a successful list request; manually added files may not be visible with the `drive.file` scope.

This is intentionally throwaway code used to answer Phase 1 blockers before implementing the real provider.

The experiment listens only on IPv4 loopback (127.0.0.1:3030). Pending OAuth logins expire after ten minutes and are capped at 128; a full store returns HTTP 429 until a callback consumes a login or transactions expire.
