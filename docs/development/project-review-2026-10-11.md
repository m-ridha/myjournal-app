# Project development review — 2026-10-11

## Follow-up status (2026-10-11)

The review below is the original assessment at a5f761f. Subsequent work completed
its focused Chrome UI checklist, reconciled current phase summaries, expanded
accessibility QA to 30 passing scans (including Terms and signed-out pages), and
added race tests plus pinned govulncheck to PR CI. See
[UI acceptance evidence](ui-acceptance-checks.md) and [progress](PROGRESS.md).
The original recommendations and verification remain historical evidence.

Remaining priorities are branding/ownership verification and support-mailbox
confirmation, followed by separately authorized reliability/performance work if
broader guarantees are needed. The deliberate reconnect action remains optional
and deferred. Restricted Google Testing acceptance and documented concurrency,
latency and revision limits remain in effect.

## Original review

Reviewed source commit a5f761f, AGENTS, progress, phase acceptance, release/Google/UI
checklists, architecture/connection/journal/cache/web code and test coverage,
release tooling and the merged public-page PR #21. This is a development-status
and maintainability review, not an exhaustive security audit or renewed live
Google/production acceptance.

## Phase assessment

| Phase | Assessment | Remaining work |
| --- | --- | --- |
| 0–1 bootstrap/spikes | Recorded acceptance complete for the adopted approach | No routine rerun needed |
| 2 core/local | Implemented with shared contracts and boundary enforcement | Preserve correctness during future changes |
| 3 UI | Implemented; full acceptance incomplete | Native 200% zoom, screen-reader announcements, remaining focus/non-text contrast, external trash-icon provenance |
| 4 Google adapter | Implemented; restricted test-user acceptance recorded | Broader publication/branding is separate; do not reopen accepted sign-in/storage/restart checks routinely |
| 5 reliability/performance | Substantial mocked coverage; accepted restricted-release limitations remain | Targeted live uncertain-write/outage cases and controlled performance evidence before broader guarantees |
| 6 release | Restricted release automation accepted | Branding ownership/approval, support mailbox confirmation, and applicable wider-audience decisions |

PR #21 is merged. Its page/tag implementation does not establish deployment,
Search Console ownership, mailbox delivery or Google approval. Consult
[branding steps](google-branding-verification.md). The documented 24-hour wait
starts after ownership acceptance, not after code merge.

Full UI acceptance remains open in [UI checks](ui-acceptance-checks.md).
The new Terms route should be included in future acceptance runs.

Historical save measurements report a 3.469-second browser median; this is not
p75 acceptance and does not meet the original subsecond save goal. The measured
external calls dominate elapsed time; more local abstraction or another cache
library is not an evidence-based latency fix. Preserve fresh conflict checks and
avoid blind mutation retries. Keep January 3 investigation/cleanup and persistent
hint caches deferred under the existing user decisions.

## Recommended improvements, in order

1. Close accessibility and branding acceptance first, preserving the existing
   restricted test-user policy until separately changed.
2. Reconcile active acceptance summaries. phase-acceptance.md still describes
   Google production registration as gated and remaining inline theme setup,
   although current composition and external scripts implement those changes.
   Separate historical observations from a concise current phase/status table;
   retain old evidence and explicit waivers rather than rewriting history.
3. Extend scripts/python/check-accessibility.py beyond its current signed-in seven-route
   set to cover /terms and the signed-out homepage/public pages in both themes.
   Existing scans cannot stand as evidence for routes they never visit.
4. Add race tests and govulncheck to PR or scheduled checks. PR checks currently
   run ordinary tests and architecture probes; vulnerability checks run during
   release. Detecting regressions earlier avoids discovering them only after
   publishing a release. These are improvements, not failures of current checks.
5. Consider a deliberate reconnect action after 401, with unsaved-text backup,
   explicit account handling and no automatic navigation. The current editor
   preserves text and displays a sign-in message; the action is already parked
   in ideas.md and is not authorized by this review.

No additional provider, Redis, frontend framework, database, offline sync or
large architecture refactor is needed to close the current acceptance work.
Supporting simultaneous external editing or multiple writers would require a
separately reviewed protocol; existing process locks/cache ports cannot provide it.

## Fresh verification

- golangci-lint fmt --diff ./...: no differences.
- golangci-lint run: zero issues; go vet ./...: passed.
- go test -race -short ./...: all packages passed.
- make test-js: all five suites passed using the available local runner.
- Deployment helper unittest discovery: three tests passed.
- make test-architecture: all 31 probes passed separately; cleanup verified.
- govulncheck v1.8.0: zero reachable/imported-package vulnerabilities, one
  module-only advisory reported without an application call path.

No new live account/browser/deployment tests, configuration changes, commits,
pushes or remote-thread actions were performed during this review.
