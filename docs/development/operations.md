# Operations

Use synthetic data for operational checks. Never record credentials, account claims, journal text, authorization codes, callback queries or whole cookies. See [SECURITY](../../SECURITY.md) for private reporting and [testing](testing.md) for regression procedures.

## Configuration

[.env.example](../../.env.example) lists runtime settings. Local defaults: port 3030, APP_BASE_URL=http://localhost:3030, PROVIDERS=local, LOCAL_DATA_DIR=./.devdata, info logging and non-Secure HTTP cookies. Listener port and base URL are separate settings; keep them aligned. `make run` loads `.env`; direct go run does not.

SESSION_KEY must encode 32 random bytes in base64. Keep it private and stable across restarts/rollback; rotating it invalidates existing encrypted sessions. Startup validates configuration before composing adapters. Never print `.env` or request decrypted Railway variables.

For localhost Google testing, use a **separate Desktop app OAuth client** in Google Console; Desktop app clients permit loopback redirects by design and do not trigger App Security warnings. Do not add loopback redirect URIs to the production Web application client; Google flags loopback redirects on production Web clients as an insecure flow vulnerable to authorization-code interception. Privately configure the dev client's GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET, PROVIDERS=google, GOOGLE_TEST_MODE=true, COOKIE_SECURE=false and the localhost base URL. Restore PROVIDERS=local and GOOGLE_TEST_MODE=false afterward. Ordinary production Google configuration uses the production client ID/secret, HTTPS, COOKIE_SECURE=true and GOOGLE_TEST_MODE=false.

## Domain and Google branding

The production origin is `https://myjournal.ink`. Home, `/privacy` and `/terms` must be readable without sign-in. OAuth's authorized domain is myjournal.ink and its exact production callback is `https://myjournal.ink/api/v1/auth/google/callback`. Keep Railway runtime APP_BASE_URL and GitHub's release health-check APP_BASE_URL aligned. Changing one does not update the other or Google Console.

Railway serves certificates for configured custom domains. A www hostname needs its own hosting/HTTPS redirect configuration and DNS; apex configuration does not establish www support. Do not assume a registrar redirect supports HTTPS without verifying it. Canonical Google login redirects use the configured origin before creating login cookies; keep old editor tabs available for recovery rather than redirecting every API/editor request.

Google branding verification and application publication are complete: the OAuth consent screen shows "myjournal" with verified branding, and the publishing status is In Production (public). Keep the landing-page google-site-verification meta tag deployed, and keep public home/privacy/terms available and support@myjournal.ink deliverable. See [status](status.md).

Request only openid, email, profile and drive.file. Do not broaden scope to discover manually created folders. With the application In Production, refresh tokens are not restricted by Testing mode's seven-day expiry, while the application maintains its fixed 30-day session cookie. Reconnection preserves text and account consistency across token refresh or expiry.

## Railway releases

Infrastructure details and safe plan restrictions belong in [.railway/README.md](../../.railway/README.md). The SDK is isolated from the application module. Variables/secrets remain unmanaged by IaC. Keep the existing production target, one Singapore replica and GitHub branch autodeploys disabled; a single replica does not prevent temporary revision overlap during deployments.

GitHub configuration:

| Setting | Location |
| --- | --- |
| RAILWAY_PROJECT_ID | Repository variables |
| APP_BASE_URL, RAILWAY_SERVICE_ID, RAILWAY_ENVIRONMENT_ID | production environment variables |
| RAILWAY_TOKEN | Secret, preferably production-environment scoped; repository secret supported |

The release path publishes a non-prerelease GitHub Release with a validated vX.Y.Z tag (optional hyphen suffix, no leading zeros in core numbers or + metadata). Tag pushes, PRs and merges alone do not deploy. Manual dispatch is not supported.

The release job checks main ancestry and code, serializes production deployment, applies a restricted pinned infrastructure plan, verifies convergence, then uploads a clean git archive with RELEASE_VERSION. The helper rejects unexpected targets, resource creation/deletion and variable changes. It polls the exact deployment ID for SUCCESS and verifies public `/healthz`. Inspect ambiguous deployments before retrying. Release publishing or live infrastructure changes require task authorization.

The Railway deployment message uses the release version; the source SHA remains traceable. Docker embeds RELEASE_VERSION into the binary/footer/log version without runtime GitHub access; local builds omit it. Never commit the archive-only version file or credentials.

## Verification and rollback

After an authorized release, verify health, canonical origin, HTTPS/HSTS, Secure/HttpOnly/SameSite cookies and synthetic sign-in/save/read/trash. Confirm logs omit callback queries and content. Check reconnect and navigation without discarding drafts. Separate automated checks from human observations in [status](status.md).

For rollback, use an authorized known-good release through the guarded release path, preserving production identity and SESSION_KEY. Inspect active deployments before retrying. Rollback does not revert Drive writes, restore trashed files or prove revision retention. Never manually apply an unrelated Railway plan as a bootstrap workaround.
