# 016: Restricted first Google production release

Status: accepted by the user on 2026-10-10; production enablement remains conditional on remaining acceptance.

## Decision

The first Google production release may use best-effort checksum conflict detection with one active app process and no simultaneous external editing. The user explicitly accepted documented lost-update risk instead of requiring verified atomic conditional updates before this restricted release.

This changes the release policy, not the storage guarantees. It does not authorize enabling production before live OAuth/Drive, HTTPS/Console and release checks pass.

## Supported use and limits

App clients handled by the same process share mutation locks. Observed stale versions conflict; current content is verified on reads, and ambiguous saves require reconciliation. External edits performed while the app is idle can be discovered by fresh reads; browsing snapshots may remain stale until their existing expiry.

Simultaneous external Drive edits are unsupported: an edit between checksum comparison and upload can be overwritten. Multiple active app processes, autoscaling, and overlapping rolling deployments are unsupported. Entry creation has no atomic per-date uniqueness guarantee across processes or external creators. Drive revisions and post-upload reads are not a substitute for atomic conflict prevention or a guaranteed backup.

## Deployment conditions

- Run one active process capable of journal writes. Include old/new deployment versions and maintenance jobs in that count.
- Stop/drain the old process before starting the new process. A maximum replica count of one or a traffic split alone is not evidence that revisions cannot overlap.
- Accept the brief downtime implied by stop-before-start deployment until a stronger protocol exists.
- Document these restrictions and residual risk for users before production sign-in is enabled.
- Keep the current production guard until the remaining gates for google production deployment are met. No new bypass flag is introduced by this ADR.

## Future change

Supporting simultaneous external edits or multiple app instances requires a separately reviewed protocol and evidence, including updates, trash and create uniqueness. Optional If-Match forwarding remains unverified against actual Drive and cannot be advertised as atomic protection. See ADR 005 for the existing checksum/version choice.
