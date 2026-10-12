# Railway preparation

Selected public origin: <https://myjournal-ink.up.railway.app>. The acceptance build is deployed. Public sign-in, synthetic save/reload/trash, post-restart read, list/calendar and sign-out have recorded evidence; Disconnect/reconnect is human-reported. See google-production-gates.md for remaining release conditions.

## Service setup

Use one Railway service, one replica in one region, no extra writer service and no persistent disk/database. railway.json builds the Dockerfile and checks /healthz. The app listens on all interfaces using Railway's PORT. Entries live in Google Drive; container data is ephemeral. Keep the accepted deployment-overlap/lost-update limits in ADR 016; one replica is not a distributed lock.

Configure Railway variables:

- `APP_BASE_URL=https://myjournal-ink.up.railway.app`
- PROVIDERS=google, COOKIE_SECURE=true, GOOGLE_TEST_MODE=false, LOG_LEVEL=info
- SESSION_KEY: stable base64-encoded 32 random bytes
- GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET: existing Google Web OAuth client

Supply secrets through Railway variables, never Git, Docker build arguments or chat. Keep SESSION_KEY stable across restart/rollback. Google Console callback is exactly <https://myjournal-ink.up.railway.app/api/v1/auth/google/callback>; preserve the four approved scopes. Keep the Console app/test-user restriction during controlled acceptance.

The acceptance build permits non-test Google registration only for an HTTPS origin with COOKIE_SECURE=true. Test mode remains loopback-only. Deploy the reviewed acceptance commit through a controlled bootstrap and complete google-production-gates.md before setting release readiness. Startup eligibility is not proof of public acceptance.

Review Railway plan, trial/credit expiry, usage limits and spend controls directly in the account. No free-tier or zero-cost guarantee is made, and this preparation does not authorize a paid plan change. Review HTTP/build/application logging and exports for OAuth query retention before actual consent; query-free application logs alone do not prove platform privacy.

## Published-release automation

Disable GitHub branch autodeploys in Railway; use the existing service with CLI uploads. No tag-push deployment is configured. GitHub publishes a non-prerelease Release with vX.Y.Z with an optional version suffix (no leading zeros in the three core numbers), checks out that tag and verifies main ancestry, then reruns required checks. CLI @railway/cli 5.64.2 is pinned.

Configure secret RAILWAY_TOKEN (project/environment-scoped deployment token), repository variable RAILWAY_PROJECT_ID, and production-environment variables RAILWAY_SERVICE_ID and RAILWAY_ENVIRONMENT_ID. IDs are configuration, not credentials. A repository token secret is supported; prefer a production-environment token secret to restrict credential availability to that environment. The repository variable RAILWAY_RELEASE_READY=true enables the accepted restricted test-user release workflow; the first release v0.1.0 passed. Restrict release/tag creation and token access.

The workflow uploads a git archive of the checked-out release commit rather than the working tree or branch head. Docker's build allowlist further limits image inputs. Git archive contains all tracked files: keep secrets, local data and private planning files untracked. Do not commit private files merely because Docker ignores them.

The helper captures the upload's deploymentId, polls that exact ID until SUCCESS, and checks public /healthz. Raw CLI output is withheld to avoid leaking diagnostics. Failed/crashed/unknown status, timeout or CLI error fails CI. An accepted upload alone is not success. The CLI-upload deployment is traced to the archive and SHA in its message; this is not an independently measured runtime commit endpoint. Confirm the deployment in Railway when investigating ambiguous outcomes or concurrent manual changes.

Do not retry an ambiguous upload blindly: it may already be deploying. Workflow concurrency prevents overlapping jobs, but an interrupted job does not cancel Railway and pending GitHub runs are not a durable release queue. Inspect the dashboard before another deploy; avoid simultaneous manual deploys.

## Rollback and remaining checks

Use Railway's rollback to a known-good deployment only after confirming its source/version and retaining environment/secrets. Verify SUCCESS, public /healthz and an existing entry/session afterward. Rollback does not undo Drive writes and retains overlap risk. If rollback is unavailable, upload the known-good source commit through the same controlled mechanism; do not redeploy latest branch head as an assumed rollback.

Pending: local Docker build/smoke (daemon unavailable), release-triggered Railway CLI deployment verification, rollback smoke, HSTS deployment verification, infrastructure logging/privacy review, and applicable Google Console/launch decisions. Public acceptance observations are recorded in findings.md; they do not prove the release workflow. The workflow remains gated and no cloud credentials were accessed.

References reviewed 2026-10-10:

- [CLI deployment](https://docs.railway.com/cli/up)
- [Deployment statuses](https://docs.railway.com/cli/deployment)
- [Config as code](https://docs.railway.com/config-as-code/reference)
- [Project tokens and CI](https://docs.railway.com/cli)

## Final readiness handoff

Before setting the repository variable RAILWAY_RELEASE_READY=true:

1. Merge the reviewed HSTS/workflow configuration changes into main and deploy that exact commit through the controlled bootstrap path. Verify /healthz returns 200 and HTTPS responses include Strict-Transport-Security: max-age=31536000. Keep one replica, branch autodeploys disabled and SESSION_KEY unchanged.
2. Confirm callback queries are absent from Railway HTTP/application logs and configured exports. Record which surfaces were checked and whether exports are enabled; dashboard observations cannot prove undocumented internal provider retention.
3. Choose the release audience explicitly. Google Console Testing restricts access to test users and normally expires this app's refresh tokens after seven days despite the 30-day session cookie. Publishing/verification is a separate human-managed process; never describe Testing as an unrestricted public launch.
4. Confirm homepage/privacy/callback configuration and any applicable Console verification. Record the scoped acceptance of measured save latency and revision-retention limitations; do not promise unmeasured retention.
5. Exercise Railway rollback to a known-good source deployment with the same secrets. Confirm health and a saved synthetic entry/session, restore the acceptance build, verify HSTS again and clean up the fixture. Rollback does not revert Drive content.
6. Confirm production environment IDs/secrets and intended tag commit, then set the repository readiness variable. Publish a vX.Y.Z GitHub Release (optional version suffix) from main. Inspect the first workflow run: code checks, exact deployment ID SUCCESS and public health must all pass. This first run validates release automation; it is not evidence available before running it. If it fails ambiguously, inspect Railway before retrying and unset readiness while investigating.

The readiness flag authorizes future published releases to deploy; it does not publish Google Console settings or create a release by itself.

## Release version metadata

The release workflow writes its already-validated GitHub release tag to RELEASE_VERSION in the source archive before upload. Docker explicitly includes that file and passes it through `-X main.version` when compiling the binary; the composition root supplies the version to the page renderer. The file is created only in the deployment archive, not stored in the repository, and optional for Docker builds; Docker leaves the version empty when it is absent. Versions are limited to empty or vX.Y.Z tags with optional version suffixes, and the footer displays them without links to the private repository. No Git metadata, GitHub credentials or runtime API request is required. This wiring takes effect only in a future authorized release containing the change; it does not redeploy the running service.
