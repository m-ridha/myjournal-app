# 016: Restricted first Google production release

Status: accepted by the user on 2026-10-10; production enablement remains conditional on remaining acceptance.

## Decision

The first Google production release may use best-effort checksum conflict detection with a single Railway service and no simultaneous external editing. On 2026-10-10 the user additionally accepted temporary instance/revision overlap and its cross-process lost-update/duplicate-entry risk, prioritizing low cost. The user explicitly accepted documented lost-update risk instead of requiring verified atomic conditional updates before this restricted release.

This changes the release policy, not the storage guarantees. It does not authorize enabling production before live OAuth/Drive, HTTPS/Console and release checks pass.

## Supported use and limits

App clients handled by the same process share mutation locks. Observed stale versions conflict; current content is verified on reads, and ambiguous saves require reconciliation. External edits performed while the app is idle can be discovered by fresh reads; browsing snapshots may remain stale until their existing expiry.

Simultaneous external Drive edits are unsupported: an edit between checksum comparison and upload can be overwritten. A single Railway service does not establish exclusive process ownership during rollout/replacement. Their locks are independent: simultaneous writes can lose updates or create duplicate entries. This residual platform-overlap risk is explicitly accepted; deliberate multi-instance scaling remains outside this restricted release. Entry creation has no atomic per-date uniqueness guarantee across processes or external creators. Drive revisions and post-upload reads are not a substitute for atomic conflict prevention or a guaranteed backup.

## Deployment conditions

- Configure one Railway service, with automatic branch deploys disabled and no artificial keep-alive traffic. Single-instance configuration is an operational control, not mutual exclusion.
- Serialize GitHub release deployments and avoid concurrent manual deployments, extra writer services and maintenance writers.
- Temporary platform/revision overlap remains possible and accepted. Do not advertise atomic conflict protection or strict single-process execution.
- Document these restrictions and residual risk for users before production sign-in is enabled.
- Use the HTTPS/Secure-cookie acceptance build for controlled public validation; keep release automation gated until google-production-gates.md is complete. No bypass flag is introduced.

Hosting choice: Railway at <https://myjournal-ink.up.railway.app>. Published GitHub Releases remain the deployment trigger. The current tag-label policy permits vX.Y.Z with optional version suffixes; GitHub releases marked prerelease remain excluded. See [Railway preparation](../railway-preparation.md).

## Initial release audience and launch limitations

On 2026-10-10 the user selected restricted test-user releases while Google Console remains in Testing, accepting the associated seven-day refresh-token expiry and the measured save-latency/revision-retention limitations. The 30-day app cookie does not extend Google's token lifetime. This is not an unrestricted public launch, a passed latency target, or a guarantee of Drive revision retention. Publishing the Console app remains a separate human-managed decision.

## Future change

Supporting simultaneous external edits or multiple app instances requires a separately reviewed protocol and evidence, including updates, trash and create uniqueness. Optional If-Match forwarding remains unverified against actual Drive and cannot be advertised as atomic protection. See ADR 005 for the existing checksum/version choice.
