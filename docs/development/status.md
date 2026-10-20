# Current status and limitations

This is a compact record of accepted evidence and remaining work, not a chronology. Requirements live in [requirements](requirements.md), decisions in [architecture](architecture.md), procedures in [operations](operations.md) and regression coverage in [testing](testing.md). Historical reports remain in Git history. Evidence below is historical acceptance, not a fresh live check.

## Accepted baseline

- Restricted Google test-user release acceptance is complete. The first published-release workflow, v0.1.0, passed on 2026-10-10. This does not mean unrestricted Google Console publication.
- Recorded automated coverage includes domain/storage contracts, synthetic OAuth/session cases, recovery/autosave/browser regressions, architecture boundaries and release helper restrictions. Synthetic checks do not establish actual Google token omission, external-writer atomicity or live latency.
- Recorded integrated/public evidence covers sign-in, synthetic save/read/trash, restart refresh, navigation/sign-out, health/HSTS, cookie attributes and rollback/restoration. Public Google Disconnect/reconnect was human-reported. A restart/read exercises refresh acquisition; actual refresh-token omission was not observed in the integrated flow.
- Focused Chrome UI acceptance includes native 200% zoom, keyboard/error focus, sampled non-text contrast and human-reported screen-reader announcements. Device-toolbar coverage was accepted instead of physical-phone testing. These checks do not establish full WCAG conformance or complete UI acceptance.
- The user reports the custom-domain HTTPS connection now works and Railway APP_BASE_URL is https://myjournal.ink. This is a human report, not a new infrastructure audit.

## Outstanding work

- Confirm deployment of the custom-domain Search Console tag, ownership verification and Google's branding decision. Previous notes recorded these as pending; page changes or successful HTTPS alone do not close them. Confirm support mailbox delivery and public policy URLs before submission.
- Align the release workflow with the published-release-only policy: it currently also permits workflow_dispatch with separately supplied ref/version. This consolidation documents the discrepancy without changing deployment behavior.
- Complete remaining broad UI/accessibility acceptance when required; preserve the explicitly accepted physical-device waiver and existing coverage limits. Automated accessibility tooling complements manual review.
- Unrestricted Google publication is a separate human decision with Console requirements; it is not implied by making source public.

## Accepted technical limits

- Google Console Testing can expire Drive refresh tokens after seven days while the app cookie lasts 30 days. Reconnection must preserve text and account consistency.
- Checksum comparison/upload is non-atomic. Same-process locks, generation guards and one replica do not eliminate external-writer or deployment-overlap lost updates/duplicate files. Avoid simultaneous external editing; do not claim multi-instance safety or deliberately scale out.
- Ordinary reads may reuse two-minute entry/absence snapshots. Cross-process/direct Drive edits can remain temporarily invisible. Content is bounded in process memory; lazy expiry does not guarantee immediate byte erasure. No production persistent journal/session store is introduced.
- Performance optimizations reduce synthetic request counts but do not establish passed live latency targets. Removed benchmark result files are not evidence available in this tree. The retained browser benchmark harness is optional tooling, not production telemetry.
- Drive may coalesce or purge revisions; no revision-retention guarantee follows from save counts or checksums.
- Draft recovery is tab-scoped and may expire on tab close. Blocked storage and suspended tabs constrain backup/notification guarantees. Keep text in the open editor when backup cannot be confirmed.
- Visible application/HTTP callback logs omit queries; no log drain was enabled in recorded evidence. Internal platform retention was not established.
