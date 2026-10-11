# S07: Encrypted cookie and Google token lifecycle

This separate spike reuses `internal/driven/session` AES-GCM encoding and keeps provider/account identity, email label, refresh token, and an existing S02 root folder ID in an encrypted HttpOnly cookie. It does not create or change Drive files. Access tokens are cached in memory under a SHA-256 hash of the refresh token, with a 128-item bound; the browser cookie is sufficient after a restart. The cache stores token values rather than a token source tied to an expired request context.

The spike uses distinct `mj_s7_session` and `mj_s7_oauth` cookies and does not change production authentication. State and PKCE verifier are encrypted in a five-minute cookie; the callback checks state, validates the ID token, and requires a verified email. A later sign-in that omits a refresh token preserves the existing token only for the same validated subject. Raw OAuth/Drive errors, cookies, tokens, account values, and callback queries are never logged or displayed.

Google's [web-server OAuth guide](https://developers.google.com/identity/protocols/oauth2/web-server) describes offline access and refresh. External behavior must still be verified through the steps below.

## Start and sign in

Stop your own previous spike with Ctrl+C. In the repository root:

```sh
set -a
source ./.env
set +a
export APP_BASE_URL=http://localhost:3030
export COOKIE_SECURE=false

go run ./spike/s07_cookie_lifecycle -root-id '<EXISTING_S02_ROOT_ID>'
```

Use the original S02 root ID, not the replacement entries/year ID. The same Google client, test account, and Console callback `http://localhost:3030/auth/google/callback` apply. `SESSION_KEY` must already contain base64 encoding of 32 random bytes. Do not print or share `.env` or the key.

Open `http://localhost:3030/` and click **Sign in with consent**. Expected callback flags: `session_cookie_set: true`, `refresh_token_returned: true`, `id_token_valid: true`, and `drive_root_read_succeeded: true`. Record `set_cookie_bytes` and cookie attribute flags. The full Set-Cookie representation must stay below 3800 bytes; record how much margin your real account/token leaves. The callback setting a cookie does not prove the browser stored it: `/check` proves that.

## Refresh, reuse, expiry, and restart

If the callback fails the root read, it reports `drive_failure_category`, `google_http_status` (0 when no API status is available), and `session_retained`. No new session is issued on that failure, so `/check` may then report `session_rejected`. Share the diagnostic flags first. Locally verify that `-root-id` is the actual original S02 root ID from the Drive folder URL, that the root is active, and that you used the same test account/OAuth client as S02. A 404 does not distinguish a missing ID from an inaccessible resource. Do not broaden OAuth scopes or create replacement folders to bypass this check.

1. Open `http://localhost:3030/check`. The cache starts empty even after login. Expect `session_cookie_valid: true`, `access_token_cache_hit: false`, `access_token_refreshed: true`, and `drive_root_read_succeeded: true`.
2. Open `/check` again. Expect a cache hit and `access_token_refreshed: false`.
3. Open `/check?force_refresh=1`. This bypasses the cached access token, seeds the token source with an expired token, and forces renewal. Expect `simulated_expiry: true`, a cache miss, refresh, and a successful Drive read.
4. Stop and restart with the exact same command and `SESSION_KEY`. Keep the same browser profile/cookie. Open `/check` **without signing in again**. Expect a cache miss, refresh, and a successful Drive read. This verifies restart with cookie-only credentials.
5. Optional comparison: on `/`, click **Sign in without forced consent**. If Google omits the refresh token, expect `refresh_token_returned: false` and `existing_refresh_token_preserved: true`; then `/check` must work. If Google returns a token, record that actual result instead.

## Tampering

On `/`, click **Tamper with this spike's cookie**. The POST flips one ciphertext byte and redirects to `/check`. Expect a redirect back to `/` showing `result: session_rejected` and the spike session cookie cleared. This intentionally invalidates only the spike session. Sign in with consent again before continuing.

## Key rotation

With a valid spike session, stop the server. In the same terminal, keep the original key in a shell variable and generate a temporary replacement without printing either:

```sh
S7_ORIGINAL_SESSION_KEY="$SESSION_KEY"
export SESSION_KEY="$(openssl rand -base64 32)"
go run ./spike/s07_cookie_lifecycle -root-id '<EXISTING_S02_ROOT_ID>'
```

Open `/check` without signing in. Expect `result: session_rejected` and the cookie cleared. Stop the server, restore the original key, and restart:

```sh
export SESSION_KEY="$S7_ORIGINAL_SESSION_KEY"
unset S7_ORIGINAL_SESSION_KEY
go run ./spike/s07_cookie_lifecycle -root-id '<EXISTING_S02_ROOT_ID>'
```

Sign in again; the cleared cookie is not restored by restoring the key. These shell changes do not edit `.env`.

## Revoked access

Run this last because it removes this client's authorization for the test account, affecting other spikes using the same client. With a valid S7 session, manually remove this app's access in your Google Account's third-party connections settings. Then open `/check?force_refresh=1` to ensure a cached access token does not hide the revocation.

Expected: refresh returns `invalid_grant`, the spike clears its session cookie, and the browser lands on `/` with `result: invalid_grant` and sign-in links. Transient refresh/Drive failures instead return a sanitized error and retain the session for retry. If the result differs, record it rather than marking revocation passed. Reauthorize afterward if you want to resume other spikes. The harness never revokes access itself.

## Browser attributes and lifetime

In browser developer tools, inspect only attributes of `mj_s7_session`: HttpOnly, SameSite=Lax, Path=/, expiry about 30 days ahead, and Secure=false in this initial run. Do not copy cookie values or whole headers. Repeat sign-in and `/check` in the browsers you intend to support, including Safari when available, recording browser names/versions and results.

For the Secure-on-localhost comparison, stop the server, set `export COOKIE_SECURE=true`, restart, and sign in in each browser. Record whether the login cookie survives the callback and whether `/check` receives the session. A failure can occur at either step. Return to `COOKIE_SECURE=false` for plain-HTTP development afterward. This is a browser experiment, not a production HTTPS substitute.

Human-reported comparison on 2026-10-08: Chrome 154.0.8037.98 accepted Secure cookies on HTTP localhost. Safari 26.4 rejected OAuth state with Secure enabled, then completed sign-in and cookie-backed refresh/Drive read with Secure disabled. Use `COOKIE_SECURE=false` for plain-HTTP localhost development; production HTTPS still requires Secure cookies and separate validation. Full evidence is recorded in `docs/development/findings.md`.

The user adopted a fixed 30-day session lifetime on 2026-10-08 (ADR 002), enforced both by cookie attributes and by `IssuedAt` on the server. Refresh does not extend the server deadline. Expired/future-issued cookies are rejected in deterministic tests; do not wait 30 days for the experiment. Production age enforcement remains pending. The lifetime may be revised to one or two weeks during development; update server-side age validation and cookie expiry together. The reused codec derives its AES key by hashing the configured secret; the spike validates the secret contains 32 random bytes in base64 and preserves the existing codec convention.

## Record evidence

Share only result flags, cookie byte count, browser versions/attribute observations, and sanitized errors. Do not share callback URLs, tokens, account claims, cookie values, screenshots containing secrets, or the root ID. Record live observations in `docs/development/findings.md` and tick S7 in `PROGRESS.md` only after review. Mock tests establish harness behavior; they are not evidence of Google or browser acceptance. The production Google provider stays disabled.
