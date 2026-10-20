# Operations

Use synthetic data for operational checks. Never record credentials, account claims, journal text, authorization codes, callback queries or whole cookies. See [SECURITY](../../SECURITY.md) for private reporting and [testing](testing.md) for regression procedures.

## Configuration

[.env.example](../../.env.example) lists runtime settings. Local defaults: port 3030, APP_BASE_URL=http://localhost:3030, PROVIDERS=local, LOCAL_DATA_DIR=./.devdata, info logging and non-Secure HTTP cookies. Listener port and base URL are separate settings; keep them aligned. `make run` loads `.env`; direct go run does not.

SESSION_KEY must encode 32 random bytes in base64. Keep it private and stable across restarts/rollback; rotating it invalidates existing encrypted sessions. Startup validates configuration before composing adapters. Never print `.env` or request decrypted Railway variables.

For localhost Google testing, privately configure GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET, PROVIDERS=google, GOOGLE_TEST_MODE=true, COOKIE_SECURE=false and the localhost base URL. Register exactly `http://localhost:3030/api/v1/auth/google/callback`. Restore PROVIDERS=local and GOOGLE_TEST_MODE=false afterward. Ordinary production Google configuration uses HTTPS, COOKIE_SECURE=true and GOOGLE_TEST_MODE=false.

## Domain and Google branding

The production origin is `https://myjournal.ink`. Home, `/privacy` and `/terms` must be readable without sign-in. OAuth's authorized domain is myjournal.ink and its exact production callback is `https://myjournal.ink/api/v1/auth/google/callback`. Keep Railway runtime APP_BASE_URL and GitHub's release health-check APP_BASE_URL aligned. Changing one does not update the other or Google Console.

Railway serves certificates for configured custom domains. A www hostname needs its own hosting/HTTPS redirect configuration and DNS; apex configuration does not establish www support. Do not assume a registrar redirect supports HTTPS without verifying it. Canonical Google login redirects use the configured origin before creating login cookies; keep old editor tabs available for recovery rather than redirecting every API/editor request.

For branding ownership, use Search Console's exact `https://myjournal.ink/` URL-prefix property with an authorized project owner/editor. Keep the landing-page google-site-verification meta tag deployed after verification. Confirm public home/privacy/terms availability and support@myjournal.ink delivery. Verify ownership, align Google branding name/home/privacy/terms, and follow Google's rejection-specific wait instructions before resubmission. Search Console ownership does not establish branding approval or authorize unrestricted publication. See [current outstanding work](status.md).

Request only openid, email, profile and drive.file. Do not broaden scope to discover manually created folders. Google Console remains Testing: Drive grants normally face a seven-day refresh-token limit despite the application's fixed 30-day session cookie. Publishing/verification is a separate human-managed decision.

## Railway releases

Infrastructure details and safe plan restrictions belong in [.railway/README.md](../../.railway/README.md). The SDK is isolated from the application module. Variables/secrets remain unmanaged by IaC. Keep the existing production target, one Singapore replica and GitHub branch autodeploys disabled; a single replica does not prevent temporary revision overlap during deployments.

GitHub configuration:

| Setting | Location |
| --- | --- |
| RAILWAY_RELEASE_READY=true, RAILWAY_PROJECT_ID | Repository variables |
| APP_BASE_URL, RAILWAY_SERVICE_ID, RAILWAY_ENVIRONMENT_ID | production environment variables |
| RAILWAY_TOKEN | Secret, preferably production-environment scoped; repository secret supported |

The release path publishes a non-prerelease GitHub Release with a validated vX.Y.Z tag (optional hyphen suffix, no leading zeros in core numbers or + metadata). Tag pushes, PRs and merges alone do not deploy. Manual dispatch is not supported.

The release job checks main ancestry and code, serializes production deployment, applies a restricted pinned infrastructure plan, verifies convergence, then uploads a clean git archive with RELEASE_VERSION. The helper rejects unexpected targets, resource creation/deletion and variable changes. It polls the exact deployment ID for SUCCESS and verifies public `/healthz`. Inspect ambiguous deployments before retrying. Release publishing or live infrastructure changes require task authorization.

The Railway deployment message uses the release version; the source SHA remains traceable. Docker embeds RELEASE_VERSION into the binary/footer/log version without runtime GitHub access; local builds omit it. Never commit the archive-only version file or credentials.

## Verification and rollback

After an authorized release, verify health, canonical origin, HTTPS/HSTS, Secure/HttpOnly/SameSite cookies and synthetic sign-in/save/read/trash. Confirm logs omit callback queries and content. Check reconnect and navigation without discarding drafts. Separate automated checks from human observations in [status](status.md).

For rollback, use an authorized known-good release through the guarded release path, preserving production identity and SESSION_KEY. Inspect active deployments before retrying. Rollback does not revert Drive writes, restore trashed files or prove revision retention. Never manually apply an unrelated Railway plan as a bootstrap workaround.
