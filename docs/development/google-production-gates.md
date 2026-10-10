# Google production enablement gates

Status: not ready. Localhost integration is implemented; the acceptance build now permits HTTPS/Secure-cookie registration, but deployment and public acceptance remain pending. This document separates live acceptance from deployment and policy decisions. No unchecked item is implied by earlier spike or mocked-test success.

## Next live session

Run the existing app on localhost:3030 in Google test mode using the existing key and configuration. Sign in with the same test account; keep unrelated editors and benchmark probes idle. Never export cookies or tokens. Use an unused past date and synthetic text only.

1. **Settings Disconnect/reconnect:** create and save a synthetic entry, then leave a second synthetic edit unsaved. Confirm Disconnect through Settings. Record successful sign-out and protected-page/API rejection; saved Drive files must remain. Sign in again with consent and reload the saved entry. Disconnect intentionally clears acknowledged sign-out backups; revoked-access recovery is a separate already partially covered flow.
2. **Omitted refresh token:** with a valid Google session, sign in without forced consent, then restart with the same key and read the entry. Successful reading after restart validates persisted refresh capability; do not assume Google omitted the token unless safe evidence establishes that fact.
3. **Folder recreation:** use a disposable year with no real entries. Save and trash its sole synthetic entry through the app. Check the year folder is empty in Drive before trashing that folder. A read must remain absent without recreating folders; the next synthetic save must recreate the year path and survive reload. Do not trash shared root/entries folders or a year containing other data. Test renamed/moved folders separately; retain duplicates and data-bearing folders.
4. **Revisions:** use the two-stage revision probe in google-live-checks.md on another empty past date. Record actual Drive Manage versions observations before/after changed and unchanged saves. Application checksum changes are not revision counts. Stop on the first error and retain the fixture for inspection; clean up only after successful verification. Do not promise retention from one observation.
5. **Uncertain-save recovery:** run the recovery checklist against Google, with synthetic text, and distinguish a rejected upload from an applied upload whose response is lost. Preserve drafts and avoid blind retries. Local browser and mocked-adapter evidence do not complete this live check.

Record date, browser/version, build, provider, synthetic fixture dates, observations and cleanup status in findings.md without account identifiers or entry bodies.

## External-writer and deployment policy: restricted release accepted

Current guarantees:

- Checksum versions detect content differences observed before upload, including stale tabs and many external edits.
- Bounded account lock stripes serialize app mutations within one process.
- Reads verify downloaded content against the observed checksum.
- Ambiguous uploads are not automatically repeated; manual retry verifies current content.

Limits:

- An external writer can modify content between comparison and upload.
- Multiple app processes do not share mutation locks. A replica count of one alone does not prove there is no overlap during rollout/restart.
- New entry creation is not an atomic per-date uniqueness operation.
- A successful read after upload does not prove an intervening external edit was never overwritten.

The user selected option 2 on 2026-10-10; [ADR 016](adr/016-restricted-google-release.md) records the accepted restriction and deployment conditions. The alternatives were:

1. **Strict concurrency gate:** keep production disabled until conditional updates are verified against real Drive, with stale preconditions rejected before mutation and fresh conditions accepted. Cover create uniqueness, trash, and multiple instances separately; update-only ETag evidence cannot close those gates.
2. **Restricted first release:** explicitly accept best-effort conflict detection, require one active app process including deployments, exclude simultaneous external editing, document residual lost-update risk, and keep production enablement conditional on all other acceptance. This is a product risk decision, not an atomicity fix.

The user accepted temporary process/revision overlap on 2026-10-10, replacing the strict one-active-process condition with a best-effort single-instance topology. Railway is the hosting target. Cross-process lost-update and duplicate-entry risk is explicitly accepted; deliberate multi-instance scaling and simultaneous external edits remain excluded. ADR 016 is authoritative. The accepted policy permits a restricted release only after remaining gates pass. The localhost test-mode restriction remains unchanged.

Source review (2026-10-10): the [Drive v3 files.update reference](https://developers.google.com/workspace/drive/api/reference/rest/v3/files/update) describes PATCH/media updates but does not establish the required conditional-upload guarantee. The [performance guide](https://developers.google.com/workspace/drive/api/guides/performance) discusses ETags conditionally and includes generic API examples; it is insufficient evidence of atomic Drive media-update support. [Revision management](https://developers.google.com/workspace/drive/api/guides/manage-revisions) is separate from application conflict detection and browser recovery.

## Production environment and release acceptance

After localhost checks and concurrency policy, verify:

- A concrete deployment topology consistent with the accepted concurrency guarantees; reproducible startup, graceful shutdown, rollback and no unintended overlapping writers.
- Production HTTPS origin, Secure cookies and callback alignment. HTTP localhost results are not HTTPS acceptance.
- Human-managed Google Console publishing, authorized redirect, homepage/privacy URLs, domain ownership and any applicable verification. Preserve the four approved scopes.
- Actual production sign-in, refresh after restart, sign-out, Disconnect, and error/recovery smoke checks with synthetic data.
- Remaining accessibility/provenance, privacy/logging, dependency vulnerability and deployment documentation gates in phase-acceptance.md.
- Performance and revision evidence reported with their actual limits. Current measured save latency does not meet the original target; record a scoped launch decision rather than silently marking it passed.

Enable automated production releases only after the applicable gates and explicit risk decisions are recorded. This checklist does not authorize publishing the Google Console app or deploying to a paid service.

Railway selected on 2026-10-10. See [preparation constraints](railway-preparation.md): one service is not evidence of exclusive writer ownership. Temporary replacement/rollout overlap remains accepted; verify infrastructure OAuth-query privacy before production enablement.

## Public HTTPS acceptance sequence

Requested by the user on 2026-10-10. Public acceptance has not started: the Railway origin is known, runtime configuration is not yet confirmed and the acceptance build is not yet deployed.

1. Use https://myjournal-ink.up.railway.app as the Railway HTTPS origin and confirm single-service configuration, stable SESSION_KEY, Google secrets, Secure cookies, no local provider and disabled branch auto-deploys. Do not send secret values to the agent. Source changes must reach a reviewed remote commit before Railway can build them.
2. Resolve bootstrap explicitly: prepare/review an HTTPS-only production registration change, retaining the localhost-only test-mode restriction and rejecting insecure production cookies. Use the Google Console app's existing test-user restriction for controlled acceptance; do not publish the Console app merely to run tests. Keep RAILWAY_RELEASE_READY unset. A separately authorized manual bootstrap deployment is needed before public checks can run; the release readiness flag is not a startup bypass.
3. Before real OAuth, request /healthz and a dummy callback with a non-secret query marker. Inspect application/platform logs and exports for that marker, without using real authorization codes. Review workspace tier and log settings. Inspect Railway HTTP/build/application logs and any exports; do not infer absent internal retention from dashboard visibility. Record the evidence and remaining limitations.
4. Check HTTPS, certificate, callback origin/path, security headers, private response cache controls and session/login-cookie attributes. Do not capture cookie values or real callback query strings in artifacts. Confirm protected API rejection before sign-in and cross-origin unsafe-request rejection.
5. Human completes Google consent with the same test account and explicit Drive permission. Verify synthetic Unicode entry save/reload, list/calendar, trash, sign-out and Disconnect. Confirm durable entries remain in Drive after Disconnect.
6. Restart/redeploy with the same SESSION_KEY, read a saved synthetic entry and verify refresh/session continuity. Test failed-save/reconnect draft preservation and conflict controls without navigating away from unacknowledged text. Reuse accepted localhost evidence where deployment does not change the behavior; record public smoke coverage precisely.
7. Clean up only synthetic fixtures, confirm Blank page after trash, record build/browser/origin and results without identifiers or content. Review remaining UI/privacy/performance launch decisions, then remove the acceptance-only restriction if appropriate and acknowledge release readiness. Public checks do not by themselves publish the Google Console app.

Public origin supplied by the user: https://myjournal-ink.up.railway.app. Initial GET /healthz returned HTTP 404 on 2026-10-10; HTTPS/OAuth acceptance has not passed.
