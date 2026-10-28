# Current status and limitations

This is a compact record of accepted evidence and remaining work, not a chronology. Requirements live in [requirements](requirements.md), decisions in [architecture](architecture.md), procedures in [operations](operations.md) and regression coverage in [testing](testing.md). Historical reports remain in Git history. Evidence below is historical acceptance, not a fresh live check.

## Accepted baseline

- Google Console OAuth application publication is In Production (public) with verified branding ("myjournal"). Restricted Google test-user acceptance previously passed with release workflow v0.1.0 on 2026-10-10.
- Recorded automated coverage includes domain/storage contracts, synthetic OAuth/session cases, recovery/autosave/browser regressions, architecture boundaries and release helper restrictions. Synthetic checks do not establish actual Google token omission, external-writer atomicity or live latency.
- Recorded integrated/public evidence covers sign-in, synthetic save/read/trash, restart refresh, navigation/sign-out, health/HSTS, cookie attributes and rollback/restoration. Public Google Disconnect/reconnect was human-reported. A restart/read exercises refresh acquisition; actual refresh-token omission was not observed in the integrated flow.
- Focused Chrome UI acceptance includes native 200% zoom, keyboard/error focus, sampled non-text contrast and human-reported screen-reader announcements. Device-toolbar coverage was accepted instead of physical-phone testing. These checks do not establish full WCAG conformance or complete UI acceptance.
- The user reports the custom-domain HTTPS connection now works and Railway APP_BASE_URL is <https://myjournal.ink>. This is a human report, not a new infrastructure audit.

- The dedicated pre-publication secret review covered publishable Git history and available GitHub-hosted material with no detected secrets; one Actions log was unavailable. This is secret-scan evidence, not a complete vulnerability audit. The user explicitly permits their GitHub username and author/committer email in history.

## Public SEO validation

The bilingual public-discovery implementation passed `make fmt lint test`, `go test -race ./internal/driver/web/...`, the editor suite under JavaScriptCore, and 24 Python helper tests. Local Chrome/axe verification covered 78 scans and 60 keyboard checks with no reported violations or page errors, including EN/ID, both themes, 320px layouts, language switching and JavaScript-disabled reading. Axe still reports incomplete checks; this does not establish complete accessibility or physical-device coverage.

The signed-out local English homepage HTML measured 10,772 bytes; the inspected deployed homepage measured 33,870 bytes. These are different builds/provider configurations, not a controlled latency benchmark. Public pages now omit editor/calendar/timezone scripts and carry only two theme messages. No field Core Web Vitals or search-ranking improvement is claimed. The read-only live check confirmed HTTP/www redirects and the existing sitemap; the old Railway hostname returned 404.

The changes remain local pending an authorized release. After deployment, follow the [Search Console checks](operations.md#search-visibility-after-an-authorized-release); no sitemap submission, indexing request, DNS change or deployment was performed here. URL and indexing contracts live in [architecture](architecture.md#public-search-discovery).

## Installability validation

Local manifest checks passed with `make fmt lint test`, separately `make test-architecture`, and `git diff --check`. Chrome on localhost loaded the manifest from both EN/ID public pages with `application/manifest+json` and decoded the supplied icons as 192x192 and 512x512. No OS installation, installed-window Google authentication, physical-device install or deployed HTTPS verification was performed. The changes remain local; follow the [installation checks](testing.md#browser-security-and-release-smoke) after an authorized release.

## Outstanding work

- ~~Confirm deployment of the custom-domain Search Console tag, ownership verification and Google's branding decision.~~ Google branding verification is complete: the OAuth consent screen displays "myjournal" and the verification status reads "Your branding has been verified and is being shown to users."
- ~~Remove loopback redirect URIs from the production Web OAuth client in Google Console.~~ The production client's loopback redirect has been removed. A separate Desktop app client is used for local testing; Desktop app clients permit loopback redirects by design and do not trigger the App Security warning. Its credentials remain local and it is never used in production.
- Complete remaining broad UI/accessibility acceptance when required; preserve the explicitly accepted physical-device waiver and existing coverage limits. Automated accessibility tooling complements manual review.
- ~~Unrestricted Google publication is a separate human decision with Console requirements; it is not implied by making source public.~~ Google Console OAuth application publication is complete and live In Production with verified branding.
- Before changing repository visibility, verify availability/enforcement of secret scanning, push protection, private vulnerability reporting and main-branch protection. Inspect remaining GitHub-hosted material, including the unavailable Actions log if recoverable, and newly published material since the completed secret review. Visibility changes require separate authorization; do not repeat completed history scanning merely because its old report was removed.
- The 85%-per-maintained-function coverage goal remains incomplete. The last recorded audit met it for 236 of 241 functions; that historical count is not a measurement of the current tree. Retain the numeric p75 latency targets in requirements without claiming they passed.
- Final footer visual checks, Docker image validation and deployed-version checks were still outstanding in the prior acceptance record. Confirm responsive alignment, release label rendering and the deployed build version before marking those gates complete.

## Accepted technical limits

- Google refresh tokens can be revoked or expire under Google OAuth policies (e.g., account password reset, 6-month inactivity or limit overruns) while the app cookie lasts 30 days. Reconnection must preserve text and account consistency.
- Checksum comparison/upload is non-atomic. Same-process locks, generation guards and one replica do not eliminate external-writer or deployment-overlap lost updates/duplicate files. Avoid simultaneous external editing; do not claim multi-instance safety or deliberately scale out.
- Ordinary reads may reuse two-minute entry/absence snapshots. Cross-process/direct Drive edits can remain temporarily invisible. Content is bounded in process memory; lazy expiry does not guarantee immediate byte erasure. No production persistent journal/session store is introduced.
- Performance optimizations reduce synthetic request counts but do not establish passed live latency targets. Removed benchmark result files are not evidence available in this tree. The retained browser benchmark harness is optional tooling, not production telemetry.
- Drive may coalesce or purge revisions; no revision-retention guarantee follows from save counts or checksums.
- Draft recovery is tab-scoped and may expire on tab close. Blocked storage and suspended tabs constrain backup/notification guarantees. Keep text in the open editor when backup cannot be confirmed.
- Visible application/HTTP callback logs omit queries; no log drain was enabled in recorded evidence. Internal platform retention was not established.
