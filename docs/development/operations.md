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

## Search visibility after an authorized release

The public search URLs are `/en/`, `/id/` and their `/help`, `/privacy`, `/terms` pages. Submit `https://myjournal.ink/sitemap.xml` in the verified Search Console property. Use URL Inspection on both language homepages and one Help page: confirm rendered content, successful fetch, declared/Google-selected canonical, and indexing eligibility. Request indexing for changed public pages when appropriate; sitemap submission and metadata do not guarantee inclusion or rankings.

Check the Page indexing and Search performance reports after Google has recrawled: compare impressions, clicks, queries, click-through rate and language landing pages over comparable periods. Inspect Core Web Vitals field reports when sufficient traffic exists; local payload reductions and lab checks do not establish field LCP/INP/CLS results. Keep legitimate links from the project's own public profiles/documentation accurate; do not buy links, fabricate reviews, add keyword-stuffed pages or publish journal content for discovery.

Track non-brand queries separately from searches for myjournal. For English pages, compare journal, diary, personal notes, private journal and safe online diary queries; for Indonesian pages, compare buku harian, jurnal, diari, catatan pribadi, jurnal pribadi and buku harian aman. Use country and landing-page filters to distinguish language audiences. Treat these as query groups to measure after recrawling, not promised rankings or an SEO score.

Maintain HTTP and www redirects to the canonical HTTPS host. The old Railway hostname returned 404 in the read-only SEO check; if historic links/indexing matter, redirect control for that hostname needs a separately authorized infrastructure change. Do not redirect active editor/recovery routes merely to improve SEO. Do not deploy, submit Search Console changes, or change DNS as part of local validation.

References: [Google AI optimization guide](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide), [Semrush SEO techniques](https://www.semrush.com/blog/seo-techniques/), [Google SEO starter guide](https://developers.google.com/search/docs/fundamentals/seo-starter-guide), [multilingual URLs](https://developers.google.com/search/docs/specialty/international/managing-multi-regional-sites), [canonical URLs](https://developers.google.com/search/docs/crawling-indexing/consolidate-duplicate-urls), [site names](https://developers.google.com/search/docs/appearance/site-names), [sitemaps](https://developers.google.com/search/docs/crawling-indexing/sitemaps/build-sitemap), [noindex](https://developers.google.com/search/docs/crawling-indexing/block-indexing), and [Core Web Vitals](https://developers.google.com/search/docs/appearance/core-web-vitals).
