# Railway preparation

Selected public origin: https://myjournal-ink.up.railway.app. Hosting configuration is preparation; no deployment or public OAuth success is claimed. Initial GET /healthz returned HTTP 404 on 2026-10-10.

## Service setup

Use one Railway service, one replica in one region, no extra writer service and no persistent disk/database. railway.json builds the Dockerfile and checks /healthz. The app listens on all interfaces using Railway's PORT. Entries live in Google Drive; container data is ephemeral. Keep the accepted deployment-overlap/lost-update limits in ADR 016; one replica is not a distributed lock.

Configure Railway variables:

- APP_BASE_URL=https://myjournal-ink.up.railway.app
- PROVIDERS=google, COOKIE_SECURE=true, GOOGLE_TEST_MODE=false, LOG_LEVEL=info
- SESSION_KEY: stable base64-encoded 32 random bytes
- GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET: existing Google Web OAuth client

Supply secrets through Railway variables, never Git, Docker build arguments or chat. Keep SESSION_KEY stable across restart/rollback. Google Console callback is exactly https://myjournal-ink.up.railway.app/api/v1/auth/google/callback; preserve the four approved scopes. Keep the Console app/test-user restriction during controlled acceptance.

The acceptance build permits non-test Google registration only for an HTTPS origin with COOKIE_SECURE=true. Test mode remains loopback-only. Deploy the reviewed acceptance commit through a controlled bootstrap and complete google-production-gates.md before setting release readiness. Startup eligibility is not proof of public acceptance.

Review Railway plan, trial/credit expiry, usage limits and spend controls directly in the account. No free-tier or zero-cost guarantee is made, and this preparation does not authorize a paid plan change. Review HTTP/build/application logging and exports for OAuth query retention before actual consent; query-free application logs alone do not prove platform privacy.

## Published-release automation

Disable GitHub branch autodeploys in Railway; use the existing service with CLI uploads. No tag-push deployment is configured. GitHub publishes a non-prerelease Release with stable vX.Y.Z (no leading zeros), checks out that tag and verifies main ancestry, then reruns required checks. CLI @railway/cli 5.64.2 is pinned.

In GitHub's production environment configure secret RAILWAY_TOKEN (project/environment-scoped deployment token), and variables RAILWAY_PROJECT_ID, RAILWAY_SERVICE_ID and RAILWAY_ENVIRONMENT_ID. Set repository variable RAILWAY_RELEASE_READY=true only after production acceptance; keep it unset for now. Restrict release/tag creation and token access.

The workflow uploads a git archive of the checked-out release commit rather than the working tree or branch head. Docker's build allowlist further limits image inputs. Git archive contains all tracked files: keep secrets, local data and private planning files untracked. Do not commit private files merely because Docker ignores them.

The helper captures the upload's deploymentId, polls that exact ID until SUCCESS, and checks public /healthz. Raw CLI output is withheld to avoid leaking diagnostics. Failed/crashed/unknown status, timeout or CLI error fails CI. An accepted upload alone is not success. The CLI-upload deployment is traced to the archive and SHA in its message; this is not an independently measured runtime commit endpoint. Confirm the deployment in Railway when investigating ambiguous outcomes or concurrent manual changes.

Do not retry an ambiguous upload blindly: it may already be deploying. Workflow concurrency prevents overlapping jobs, but an interrupted job does not cancel Railway and pending GitHub runs are not a durable release queue. Inspect the dashboard before another deploy; avoid simultaneous manual deploys.

## Rollback and remaining checks

Use Railway's rollback to a known-good deployment only after confirming its source/version and retaining environment/secrets. Verify SUCCESS, public /healthz and an existing entry/session afterward. Rollback does not undo Drive writes and retains overlap risk. If rollback is unavailable, upload the known-good source commit through the same controlled mechanism; do not redeploy latest branch head as an assumed rollback.

Pending: Docker build/smoke (local daemon unavailable), authenticated Railway CLI/deployment test, public logging/privacy review, acceptance-build deployment and public OAuth/session/recovery acceptance. The workflow remains gated and no cloud credentials were accessed.

References reviewed 2026-10-10:

- [CLI deployment](https://docs.railway.com/cli/up)
- [Deployment statuses](https://docs.railway.com/cli/deployment)
- [Config as code](https://docs.railway.com/config-as-code/reference)
- [Project tokens and CI](https://docs.railway.com/cli)
