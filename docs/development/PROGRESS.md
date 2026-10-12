# Progress

## Current status (2026-10-12)

This summary supersedes pending language in the dated evidence below; those records describe what was known at the time, not new work to repeat.

- Google OAuth/Drive registration and storage are implemented. Non-test registration is allowed with a valid HTTPS origin and Secure cookies; GOOGLE_TEST_MODE remains restricted to exact loopback/localhost URLs. Restricted Google Testing acceptance and the first v0.1.0 published-release deployment passed on 2026-10-10 (ADR 016). This is not unrestricted Console publication or branding verification.
- Theme initialization is already externalized to assets/static/js/theme-init.js, loaded before CSS. Same-origin CSP/security headers and host-only HTTPS HSTS are implemented (ADR 012). There is no remaining inline-theme migration task.
- Focused Chrome UI acceptance is complete with separately recorded human announcement evidence and device-testing waiver; coverage limits remain in [UI acceptance checks](ui-acceptance-checks.md). Full WCAG certification and blanket Phase 2/3 completion are not asserted.
- Google branding/ownership submission remains separate; see [branding verification](google-branding-verification.md). Accepted latency/revision-retention limits and non-atomic external-writer/deployment-overlap risks remain documented rather than treated as passed targets. No new Google experiments or deployment are implied by this reconciliation.

- English and Bahasa Indonesia are implemented with public language preferences, a globe/code header pill, an always-visible Settings gear and Help; Privacy is reached from Help. Account actions still require a valid session. Earlier English-only and conditional-Settings notes are historical.
- Current footer/release/logging and date-format behavior is consolidated in [the implementation summary](#footer-release-metadata-and-date-formatting-2026-10-12). Final footer visuals and Docker/deployed-version checks remain unverified; no new release is implied.

## Phase 0: Bootstrap

- [x] Create project skeleton and Go module wiring
- [x] Add app config and default server boot
- [x] Add health check route via chi
- [x] Add baseline Makefile and lint/test configuration
- [x] Add documentation and ADR scaffold
- [x] Run the full repo validation and fix any remaining lint or build issues

Validation result: `make fmt lint test` passed successfully in the current workspace.

## Phase 1: Google spike

- [x] Add Google OAuth config structure for the provider seam
- [x] Add a local dev provider and registry boundary
- [x] Create the PKCE Google OAuth spike in `spike/s01_google_oauth`
- [x] Complete live Google consent/sign-in and make a Drive API list call with a test account (human-reported 2026-10-01)
- [x] S1: refresh token present, simulated-expiry renewal succeeded, renewed token listed Drive, and ID token validated with `sub` and `email` (human-reported 2026-10-01)
- [x] Compare a subsequent sign-in without forced consent: no refresh token returned; ID-token validation succeeded (prior human report recorded in AGENTS.md)
- [x] Add the S02 folder/appProperties experiment harness, including path recreation mode, and run guide
- [x] S2: create app-tagged root/entries/year folders and verify metadata lookup survives root rename (human-reported 2026-10-01)
- [x] S2: verify manual-item visibility, trash detection, move behavior, and safe path recreation (human-reported 2026-10-08)
- [x] S02 review fixes: retain visibility mismatch flags in callback failure logs and escape backslashes/apostrophes in Drive queries; mocked regression checks and `make fmt lint test` passed (2026-10-08)
- [x] Add S04 conflict-field experiment and manual Drive rename run guide in `spike/s04_drive_conflicts`; mocked checks and `make fmt lint test` passed (2026-10-08)
- [x] S4: live content-edit/stale-save and API/manual rename comparisons passed for the test account; selected `md5Checksum` in ADR 005 (human-reported 2026-10-08)
- [x] S7: prepare `spike/s07_cookie_lifecycle` with mocked lifecycle checks and a live run guide; `make fmt lint test` passed (2026-10-08)
- [x] S7: live sign-in, cookie-backed refresh/cache reuse, simulated expiry, cold-cache restart, tamper/key-rotation rejection, sign-out, refresh-token preservation, and revoked-access handling passed for the test account; cookie header measured 584 bytes (human-reported 2026-10-08)
- [x] S7: Chrome cookie inspection confirmed HttpOnly, SameSite=Lax, Secure off, path `/`, and 30-day expiry; user adopted a fixed 30-day session lifetime in ADR 002 (2026-10-08)
- [x] S7: record tested browser as Google Chrome 154.0.8037.98 (Official Build, arm64; human-reported 2026-10-08)
- [x] S7: Chrome 154.0.8037.98 accepted Secure cookies on HTTP localhost; sign-in, refresh/Drive read, and browser Secure attribute inspection passed (human-reported 2026-10-08)
- [x] S7: Safari 26.4 completed sign-in and cookie-backed refresh/Drive read with Secure disabled; recorded `COOKIE_SECURE=false` as the HTTP localhost workaround (human-reported 2026-10-08)
- [x] S7: accept the tested cookie/token lifecycle and fixed 30-day lifetime, with the documented Safari localhost workaround; production HTTPS behavior remains unverified
- [x] Record user go decision to continue implementation (2026-10-08); remaining acceptance gates stay explicit
- [x] Prepare S03 concurrent-folder harness, deterministic race/oldest-selection tests, and live run guide in `spike/s03_folder_concurrency`
- [x] S3: five naive workers created five folders per role and returned different paths; locked workers created one folder per role and shared one path (human-reported 2026-10-08)
- [x] Record conservative S5/S6 fallback plans, provisional S11 official-doc research/release dependencies, and S12 supporting-library choices in `docs/development/phase-1-review.md`
- [x] Record Phase 1 findings and user go decision, with S5/S6 fallback plans and provisional S11/S12 decisions in `docs/development/phase-1-review.md`; release checks remain explicit

## Phase 2: Core and local provider

- [x] Local storage and day entry behavior are working for the development provider
- [x] Future dates are rejected at the router and API layer
- [x] Resolve browser IANA `mj_tz` once per request with UTC fallback; align pages, entry operations, and calendar defaults/markers (ADR 010)
- [x] Keep selected entry URLs/draft keys fixed across midnight; cover midnight, extreme offsets, DST, and browser bootstrap behavior with deterministic tests
- [x] User confirmed `/entries/today` immediately becomes `/entries/2026-10-08` (2026-10-08; browser unspecified)
- [x] User confirmed reloading a past entry preserves its selected date and the calendar marks October 8 as today (2026-10-08; browser unspecified)
- [x] User confirmed at 00:00 WIB on 2026-10-09 that the entry date became `2026-10-09` and the calendar marked October 9 as today (browser/navigation method unspecified)
- [x] User accepted selected-date preservation based on the observed `PUT /api/v1/entries/2026-10-09` request; automated tests separately cover saving across midnight (2026-10-09)
- [x] User reports Safari checks and simulated mobile screen sizes look good (2026-10-09; exact scenarios/version unspecified)
- [x] User confirmed stale timezone cookie corrects to `Asia/Jakarta` and `/entries/today` selects the correct Jakarta date (2026-10-09; browser unspecified)
- Actual phone-browser testing skipped by user decision (2026-10-09); browser device-toolbar coverage accepted for this stage
- Configurable calendar week start deferred as nice to have by user decision (2026-10-09); Monday first retained, idea recorded in `docs/development/ideas.md`
- [x] Bound pagination to pages 1–10000 and sizes 1–100; retain only page-size-plus-one dates, reject invalid API ranges with 400, and test cross-year/empty/oversized requests (2026-10-09)
- [x] Entry list and preview generation are implemented
- [x] Preview logic trims leading blank lines and uses the first meaningful line
- [x] 404 routes render the branded not-found template rather than raw Go output

## Phase 3: UI and polish

- [x] Consolidate the active PNG favicon under `assets/static/favicons/` and remove unused SVG/mask assets (2026-10-08).
- [x] Add the user-provided, light/dark adaptive SVG favicon to embedded static assets and the shared page layout (2026-10-08).
- [x] Add a 64×64 transparent PNG favicon fallback derived from the SVG after Safari 26.4 displayed a localhost letter fallback; browser confirmation remains pending (2026-10-08).
- [x] Add Safari pinned-tab mask SVG and fresh icon URLs after the PNG fallback did not resolve Safari's missing normal-tab icon; Safari checks remain pending (2026-10-08).
- [x] Confirm Safari displays the same PNG in an isolated PNG-only page; simplify the app to one PNG icon link at a fresh filename after the mixed icon setup failed. Safari app icon at `localhost:3030` confirmed after rebuilding its favicon cache; pinned-tab recheck remains pending (2026-10-08).
- [x] Editor page and date nav are wired to the local API
- [x] Branded 404 page is connected to the router and the app shell
- [x] Save status styling is visible and resets to neutral when the user edits again
- [x] Save text uses the shorter status wording: `Saved`
- [x] Draft persistence keeps unsaved text locally if a save fails or the page is left mid-edit
- [x] Page, browser interaction, accessibility, and API error copy use the English locale catalog
- [x] Private pages redirect without a valid session; entry APIs reject missing or invalid sessions
- [x] Services are constructed from the decoded session provider/account rather than a fixed local connection
- [x] Startup rejects invalid environment values, malformed session keys, unknown providers, and non-loopback local mode
- [x] Initial production Google rejection guarded pending live acceptance (2026-10-10); subsequently superseded by HTTPS-origin/Secure-cookie validation and restricted release acceptance. GOOGLE_TEST_MODE remains localhost/loopback-only.
- [x] Fix draft recovery, snapshot-based save acknowledgement, dirty tracking, empty-edit backup, failed-save retry, and guarded storage access; add account-scoped drafts and explicit legacy recovery (2026-10-08)
- [x] Wire Settings sign-out with account-scoped/legacy draft cleanup and cross-tab notification; account scope excludes labels and credentials
- [x] User confirmed Failed logout recovery browser checklist works (2026-10-09); commit/push to PR remains user-managed
- [x] Fix PR #3 failed-logout finding: preserve backups and editor usability until logout succeeds; distinguish/retry cleanup failure after logout; pending/503/network/retry regressions pass (2026-10-09)
- [x] Add dependency-free editor interaction regressions to `make test` and Go coverage for draft-scope isolation/rendering; `make fmt lint test` passed
- [x] User reported editor-recovery checks 1–6 and sign-out/theme-retention check 8 passed (2026-10-09); full conflict actions remain pending
- [x] Lock initial editor typing until GET succeeds, including pre-JavaScript HTML; retain Retry on failure and typing during PUT; automated checks passed
- [x] User confirmed check 7: accurate unavailable-backup warnings, successful online save, and preserved visible text after offline failure (2026-10-09)
- [x] User confirmed check 9: explicit older-draft review/cancel/restore and account-scoped recovery after reload (2026-10-09)
- [x] User confirmed check 10 initial-load lock and continued typing during slow saves; newer text shows Unsaved changes after acknowledgement (2026-10-09)
- [x] Extract all three inline template SVGs into `assets/static/svgs`; CSS masks retain theme colors and accessible labels (2026-10-09)
- [x] Remove Settings Appearance by user decision; retain shared header theme toggle and add signed-in Settings gear link using shared theme colors and localized accessible label (2026-10-09)
- [x] Add conditional Keep mine, confirmed Load the other version, and review-before-save Keep both; stale recovered drafts offer conflict actions
- [x] Isolate new drafts in account/date-scoped sessionStorage by user decision; preserve reload recovery and explicit migration of old local backups (ADR 011)
- [x] User confirmed all eight conflict/tab-isolation/Settings browser checks passed (2026-10-09): independent reload backups, retained conflict actions, all resolution choices, delayed/failed resolution, sign-out cleanup, and header navigation/theme
- [x] Make local saves atomic, serialize version checks/mutations across store instances, and preserve all repeated-trash copies; automated failure/concurrency/cancellation checks and local race detector passed (2026-10-09)
- [x] Validate UTF-8 in domain/local storage and raw JSON (including unpaired surrogate escapes); preserve invalid external files and propagate preview failures (2026-10-09)
- [x] Make previews UTF-8-safe within byte caps, trim leading whitespace before truncation, and retain ellipsis through list lookahead; extend shared contracts to cached local storage (2026-10-09)
- [x] Enforce fixed 30-day server-side session age and matching persistent-cookie expiry; reject future/missing issuance and old-key cookies (2026-10-09)
- [x] Add standard-library cross-origin protection for unsafe requests, localized rejection, 1 MiB save-envelope bounds, and query-free request logging (2026-10-09)
- [x] Fix PR #5 review: show localized HTTP 413 size-limit/copy-before-shortening guidance; regression checks preserve newer edits, reload recovery, retry after shortening, and accurate unavailable-backup warnings (2026-10-09)
- [x] Add HTTP read/write/idle deadlines and wait for graceful request draining before cleanup; regenerate Wire and cover lifecycle behavior (2026-10-09)
- [x] Extract theme bootstrap and enforce same-origin CSP, nosniff, framing/referrer protections, and no-store for dynamic responses; Chrome/native Safari browser flows verified with documented limits (ADR 012, 2026-10-09)
- [x] Run CSP/browser QA against localhost: Chrome 155 and native Safari 26.4 pass the covered editor/theme/list/calendar/conflict/logout/header flows and inline-script enforcement probe; Safari full startup-console capture remains unverified (see csp-browser-qa.md, 2026-10-09)
- [x] Align API JSON errors, wrapped reconnect/unavailable mapping, required save strings, missing-entry semantics, calendar validation, and cookie authentication documentation; regenerate OpenAPI (2026-10-09)
- [x] Address PR #7 review: retain Allow headers on custom API/page 405 responses using registered router matches; cover exact method sets (2026-10-09)
- [x] Add safe five-second idle autosave, unchanged suppression, one-flight acknowledged follow-ups, failure pauses, recovery/conflict review gates, and best-effort blur/page-hide flushing (ADR 013, 2026-10-09)
- [x] Add confirmed entry trash with pending-mutation locks, failure recovery, acknowledged draft cleanup, and Newer/Older list links with bounded page queries/retry (2026-10-09)
- [x] Move trash action to a theme-aware accessible SVG button inside the editor, with a solid floating background, right-side text clearance, keyboard focus and touch visibility; Chrome desktop/touch emulation checked (2026-10-09)
- [x] Address PR #8 review: reset saved styling on confirmed deletion; cover HTTP failures/lost acknowledgements with retained recovery and neutral status (2026-10-09)
- [x] Ignore stale calendar successes/errors and expose loading state; implement confirmed Settings disconnect through provider revocation, retaining session/drafts on failure and preserving saved files (local Chrome QA and automated regressions, 2026-10-09)
- [x] Verify and fix depguard path matching with seven temporary forbidden-import probes; add make test-architecture and explicit clock wiring boundary (2026-10-09)
- [x] Follow device theme until explicit selection, tolerate blocked/invalid theme storage, add reduced-motion support and localized navigation/calendar status semantics; focused Chrome keyboard/reflow checks pass (2026-10-09)
- [x] Adopt JSON API and vanilla JavaScript; exclude HTMX by user decision, record ADR 014, and preserve migration research as historical documentation (2026-10-09)
- [x] Fix dark Disconnect contrast and calendar group naming; add optional repeatable accessibility QA with 18 passing scans, arrow contrast measurements, and keyboard reachability/focus checks (2026-10-10)
- [x] Compare Monochrome source with current UI and local history; document visual inspiration rather than a copied theme port in ADR 004, preserving comparison limitations and separate icon-source follow-up (2026-10-10)
- [x] Focused Chrome UI checklist: native 200% zoom, keyboard/error-state focus, sampled non-text contrast, human-reported screen-reader announcements and external trash-icon provenance have recorded evidence (2026-10-11; see ui-acceptance-checks.md). This does not mark full Phase 2/3 acceptance or accessibility certification complete.

## Restructuring addendum

- [x] Generated OpenAPI code moved to `internal/web/api`
- [x] `journal.Store` and `journal.Clock` moved into `internal/journal/store.go`
- [x] Entry listing, calendar, open, save, and delete operations now go through `journal.Service`
- [x] Local provider is covered by the shared `journal/storetest` contract suite
- [x] Local provider now rejects stale saves after an entry was deleted
- [x] Wire supplies the clock from the composition root and generated output is synchronized
- [x] Wire creates one shared in-memory cache with TTL, LRU capacity, and account-scoped keys
- [x] Pagination fetches previews only for the requested page and stops after it can determine `HasNext`
- [x] Calendar entry markers load from one year index instead of reading every day individually
- [x] Locale catalog is loaded by `internal/i18n` and shared with templates and browser scripts
- [x] ADRs 008 and 009 document generated API placement and Wire
- [x] `internal/web/middleware` contains focused request time-zone resolution with tests; other middleware concerns remain pending

## Current limitations and follow-ups

- Focused Chrome UI checks are complete with recorded limitations and human-reported announcement evidence; full Phase 2/3 completion and accessibility certification are not asserted. See [UI acceptance](ui-acceptance-checks.md).
- Google Console remains Testing. Restricted test-user releases are accepted; unrestricted publication/verification is not claimed.
- Checksum comparison/upload and entry creation are non-atomic across external writers/processes. ADR 016 records accepted platform-overlap risk; deliberate multiple writers remain unsupported.
- Exploratory latency and Drive version measurements are recorded with limits. Original performance targets are not marked passed; the user accepted the gap for this restricted release.
- January 3 failure investigation/fixture cleanup and optional performance experiments remain deferred by user decision.
- The first published-release workflow, public OAuth/session/storage checks and rollback passed. No active Railway activation blocker remains for the restricted audience. Visible logging was checked and no drain is enabled; undocumented internal hosting retention is not proven.
- Physical-phone testing is waived in favor of device-toolbar coverage; native Safari startup-console history remains unverified.

## Notes

This project intentionally follows the implementation guide and keeps the Google live verification steps for the human-assisted Phase 1 tasks.

The local development provider is now sufficient to validate the journaling flow, route semantics, API contract, and UI behavior without needing live Google access. The app currently behaves as intended in local mode:

- signed-out landing flow
- date-specific entry pages
- future-date rejection
- entry preview generation from the first meaningful line
- save state transitions (`Saved`, `Unsaved changes`, `Saving…`)
- branded not-found page rendering

OpenAPI base contract added: the API definition now lives in `docs/api/openapi.yml`, and the generated chi-server scaffold is emitted under `internal/web/api/` using `oapi-codegen`.

## Contributor onboarding

- [x] Reconcile root `AGENTS.md` with current code, accepted Google/editor/date/conflict/logout observations, and user deferrals; distill product/UI/phase requirements so private planning files are not required (2026-10-09)

- [x] Address PR #4 onboarding review: link README to the tracked guide, shorten always-loaded instructions, and preserve detailed requirements in `docs/development/phase-acceptance.md` (2026-10-09)

## Pull request automation

- [x] Add PR creation/update checks for formatting drift, lint, all JavaScript/Go tests, and sequential architecture probes; validate workflow locally (2026-10-09). First hosted run passed all steps (run 37961548230, inspected 2026-10-09).

## Google integration implementation (2026-10-10)

- [x] Code exchange with PKCE, Google signature/audience/issuer/expiry/nonce checks, required Drive grant, verified email, and same-account reconnect binding
- [x] Compact versioned encrypted-cookie credentials; preserve omitted refresh tokens for the same account; bounded access-token cache and refresh/revocation error handling
- [x] Drive Store methods with shared account lock stripes, tagged/rename-stable folder discovery, write-only recreation, deterministic duplicate preservation, complete pagination, bounded previews/read retries, checksum versions, and trash
- [x] Explicit localhost-only GOOGLE_TEST_MODE and existing-folder Settings link; normal production guard retained
- [x] Homepage offers only configured sign-in providers; live Chrome Google button navigation and local/Google/both regression checks passed (2026-10-10)
- [x] Shared mocked Drive contracts, per-request concurrent creation, signed synthetic identity checks, credential/revocation/failure regressions
- [x] Human confirmed existing Console callback, Drive API, and test-user prerequisites (configuration not independently inspected)
- [x] Partial integrated Chrome smoke: user sign-in, synthetic Unicode save/reload, previews/calendar, stale-tab conflict/recovery, Keep both, Settings folder link, and acknowledged trash/removal (2026-10-10)
- [x] Integrated Keep mine and Load the other version (cancel/confirm); non-forced-consent reconnect and subsequent Drive read (2026-10-10; omitted refresh token not independently observed)
- [x] User-assisted same-key restart followed by observed signed-in Drive reload; October 2 synthetic fixture trashed and absence verified (2026-10-10)
- [x] External Drive Manage versions upload detected by stale-save conflict; local draft retained and Keep both retrieved external text (2026-10-10; check/upload race still open)
- [x] User-revoked Google connection retained both draft versions; fresh reconnect, original-tab Save (200), reload, and synthetic cleanup passed (2026-10-10; first callback rejection remains unexplained)
- [x] Covered integrated live sign-in/restart/storage/reconnect/conflict smoke checks completed; later Google/recovery and Railway evidence below closes the restricted-release gate. This is not exhaustive real-outage or external-writer coverage.
- [x] Record actual changed/unchanged Drive version observations (50 changed, no additional versions after 50 unchanged saves) and exploratory live latency. Controlled cold/warm cache state remains unverified; user accepted measured latency and revision-retention limits for the restricted release.
- [x] Resolve the release policy in ADR 016: restricted single-service topology, accepted temporary overlap/lost-update/duplicate risk and no simultaneous external edits. The non-atomic storage race is not fixed.

The later live Google and Railway sections record completed restricted-release acceptance. Exhaustive provider reliability and unrestricted production publication are not claimed; earlier spike evidence retains its original scope.

## Go review follow-up (2026-10-10)

- [x] Review Go style and clean architecture; record prioritized findings and fresh checks in [Go code review](go-code-review.md)
- [x] Address cache/date/path/logger issues, shared provider constants, explicit configuration, shared session policy, Settings/rendering/cancellation handling, safe auth diagnostics, and batched Drive previews; record conventions in AGENTS.md
- [x] Verify refactor with make fmt lint test, all seven architecture probes, go vet ./..., and go test -race -short ./... (2026-10-10)
- Deferred: Optional local bounded-prefix previews remain deferred, not an unfinished release prerequisite.

## Performance baseline (2026-10-10)

- [x] Add isolated real-adapter/service/cache benchmarks against fake Drive; run six repetitions of 20 operations for eight scenarios and retain sanitized raw output/source manifest
- [x] Confirm request counts and profile changed-save allocations; record [baseline findings](performance-benchmark.md) without changing production behavior
- [x] Add same-origin browser probe with overwrite refusal, stop-on-error, sanitized reporting, and recoverable cleanup; safety regressions pass
- [x] make fmt lint test, seven architecture probes, benchmark race smoke, and git diff --check pass
- [x] Collect user-executed live browser probe timings: 60 HTTP-200 samples, reported cleanup verified; post-save browse median 7.078 s (user confirms January 2 invocation but result labels January 1; mismatch unexplained; running build/cache state unverified)
- [x] Implement and separately measure upload streaming, operation-scoped discovery, and targeted invalidation; production concurrency/release gates remain unchanged

- [x] Retain January 2 successful results and January 3 partial failure; keep failed sample and unknown write outcome explicit
- [x] Add probe progress/single-running guard and reproduce/fix deadline errors falling through to HTTP 500 (now existing 503 unavailable response)
- Deferred by user: January 3 error-category investigation. Read-only fixture inspection is complete; retain the fixture without cleanup.

## Performance candidates (2026-10-10)

- [x] Revise private performance plan and add tracked implementation plan/ADR 015
- [x] Remove 16 MiB SDK upload buffering with bounded single-request streaming; changed-save allocations ~17.04 MB → 0.17 MB
- [x] Add bounded per-operation discovery reuse through an optional Store port and cache forwarding; cold page calls 28 → 24
- [x] Target preview/index invalidation while preserving TTL, account isolation and generation barriers; off-page save/page calls 33 → 5, visible save/page 29 → 9 against matching scoped baseline
- [x] Retain per-candidate raw benchmark runs; add visible/off-page comparison and focused integrity/race/error regressions
- [x] make fmt lint test and seven separate architecture probes pass; go vet ./... and go test -race -short ./... pass
- [x] Record user-assisted targeted live comparison: 30 HTTP-200 samples, cleanup verified; post-save browsing median 6.673 → 2.277 s (~66% lower), changed save 3.385 → 3.632 s (no improvement); exploratory limits and failure-rate acceptance remain explicit
- [x] Inspect retained January 3 fixture: user-reported two stable reads match the final attempted B content; no cleanup performed, failure cause remains unconfirmed

## Save latency investigation (2026-10-10)

- [x] Add debug-only sanitized Drive transport timing categories, header/body durations and outcomes; preserve request behavior and no raw URL/error logging
- [x] Add read-only January 3 fixture inspector returning only stability/known-synthetic-content flags; no cleanup or write
- [x] Fresh make fmt lint test, Google package race tests and git diff --check pass
- [x] Record human-assisted January 3 inspection flags: final attempted content is present despite the failed response
- [x] Collect January 5 live timing records: ten successful five-call saves; upstream calls account for 99.94% of aggregate server time (discovery 40.5%, metadata 9.9%, upload 49.5%); cleanup verified
- [x] User deferred January 3 root-cause investigation; retain evidence and fixture without cleanup
- [x] Evaluate and implement fresh-read uncertain-save reconciliation with Google live acceptance; record restricted concurrency policy in ADRs 013/016. No historical-outcome or atomicity guarantee.

## Uncertain-save retry (2026-10-10)

- [x] Implement approved fresh-read-before-manual-retry behavior with exact snapshot acknowledgement and original-version conditional retry
- [x] Add lost-acknowledgement, later-typing, failed-read, conflict, empty/absent, reload and sign-out regressions
- [x] Document current-content versus historical-outcome limits in ADR 013 and recovery checklist; preserve external-writer production gate
- [x] Fresh make fmt lint test passes (zero lint issues, five JavaScript suites, all Go packages); git diff --check passes
- [x] Agent-driven Chrome/local-provider uncertain-save retry checklist passes with scoped acknowledgement-loss/delay/network-failure injection; sign-out race and recoverable fixture cleanup verified (2026-10-10). Safari/Google coverage not implied

Performance plan status refreshed (2026-10-10): focused iteration complete; private plan labels the original proposal as historical, records completed measurements/changes, and separates optional controlled-session evidence and deferred experiments from production gates.

- [x] Review and adopt dependency-free shared test origin fixtures with standard Go naming; retain independent contract/boundary literals and production/test separation.

## Empty-entry status correction (2026-10-10)

- [x] Returning an unsaved new entry to empty text restores Blank page; an existing empty file retains Saved.
- [x] Regression reproduces the original failure and verifies no autosave PUT or retained backup after reverting; make fmt lint test passes.

## Google reliability regression coverage (2026-10-10)

- [x] Verify safe-read retry classification: 429, 5xx and rate-limited 403 retry; authentication, permissions, absence, precondition and unclassified transport errors return directly.
- [x] Verify three-attempt exhaustion, recovery within that bound, and cancellation/deadline during backoff using testing/synctest without wall-clock sleeps.
- [x] Simulate Drive applying both new-entry and update uploads before returning 503: report unavailable without automatic upload retry, recover applied content/version through Get, and reject stale original-base retries.
- [x] Fresh make fmt lint test, go test -race ./internal/provider/google, and git diff --check pass. These are synthetic reliability results, not live Google or production acceptance.

Historical next-step note superseded: Disconnect, sign-in/restart refresh capability, disposable-year recreation and revision observations were subsequently completed; release policy and restricted Railway release acceptance are recorded below. Actual refresh-token omission during integrated live sign-in was not observed.

## Production Google acceptance preparation (2026-10-10)

- [x] Prepare [production gates](google-production-gates.md), with synthetic-only Disconnect, reconnect, empty-year recreation, revision and uncertain-save checks.
- [x] Recheck official Drive update/performance guidance; no atomic conditional media-update guarantee established.
- [x] Settings Disconnect/reconnect: user confirmed sign-out, missing Drive consent was correctly rejected (grant/401), explicit consent succeeded, and agent reload verified saved synthetic file preservation.
- [x] Integrated disposable-year recreation: absent read did not recreate folder; save recreated it and survived reload. Both synthetic 2019-02-20 files trashed through Drive; editor absence verified; empty replacement year retained.
- [x] Changed revision stage on 2019-02-21: user reported 50 successful saves/50 checksum changes in 228968 ms; agent observed 50 Drive versions (current plus 1–49).
- [x] Unchanged revision stage on 2019-02-21: user reported 50 successful saves/0 checksum changes in 120301 ms, no uploads in supplied logs; agent reopened Drive Manage versions and observed the same 50 versions.
- [x] User trashed synthetic 2019-02-21 through the app and confirmed reload shows Blank page; revision fixture cleanup complete.
- [x] User accepted restricted first-release policy (ADR 016): one active app process, no simultaneous external edits, explicit lost-update risk. Deployment compliance and remaining release acceptance stay open.

This section records scoped live observations; later Railway acceptance establishes restricted test-user release readiness, not unrestricted production publication.

- [x] Run dependency vulnerability baseline with govulncheck v1.8.0 on Go 1.27.1.
- [x] Upgrade to Go 1.27.2 and patched x/net dependency family; govulncheck v1.8.0 reports zero reachable/imported-package vulnerabilities. Unused openpgp module-only advisory remains documented.
- [x] Update CI linter to v2.14.0 for Go 1.27.2 compatibility; formatting, lint, JavaScript/Go tests and seven architecture probes pass.

- [x] User confirmed Google sign-in without forced consent followed by same-key/config restart and successful existing-entry reads. Actual OAuth refresh-token omission was not observed.

- [x] Human Google-backed synthetic lost-acknowledgement test: failed feedback retained draft; manual retry GET reconciled to Saved without a second PUT (2019-02-22).
- [x] Complete human Google uncertain-save cases and clean up the synthetic fixture; see findings for scope and storage-inspection limits.

- [x] Human Google uncertain-save newer-text case: paused autosave, verification GET then stale-base PUT/409, retained text/conflict controls, Keep mine PUT succeeded.

- [x] Human Google recovery GET checks: typing during delayed verification passed; offline verification issued no PUT, and online manual retry saved successfully.
- [x] Human Google sign-out during pending recovery GET: sign-in feedback, only GET/no follow-up PUT; reconnect retained text and showed Saved. Storage cleanup was not independently inspected.
- [x] User confirmed failed-save pause does not automatically retry; trashed synthetic 2019-02-22 and reloaded to Blank page.

- [x] Replace raw chi panic logging with generic event/request ID recovery; preserve API JSON errors, abort semantics and committed-response safety. Fresh make fmt lint test passes.
- [x] Complete startup/server/error privacy review and restricted-release HTTPS/deployment acceptance; sanitized panic/server diagnostics, public headers and first automated release are recorded below.

- [x] Review app startup/server logging; sanitize net/http diagnostic fallback, regenerate Wire and verify secret-omission regression. Fresh make fmt lint test passes.
- [x] Select Railway and verify HTTPS/Console test-user setup and visible query-free logging with no drains. Temporary rollout overlap is explicitly accepted (ADR 016), not proven absent; unrestricted Console publishing remains outside this release.

### Issues for Branding verification attempt in GCP

<https://console.cloud.google.com/auth/branding?project=myjournal-app-509305>
reference: <https://support.google.com/cloud/answer/13464321?sjid=12620915019931986057-NC>

- [ ] The website of your home page URL "<https://myjournal-ink.up.railway.app>" is not registered to you.
Verify ownership of your home page, then wait 24 hours before retrying to allow our systems to update.
- [x] Your privacy policy page at "<https://myjournal-ink.up.railway.app/privacy>" does not have sufficient content.
We were unable to confirm your app's compliance. Please update your privacy policy to clearly and sufficiently detail your app's data collection and usage, then retry.
- [x] Your home page is behind a login page.
We were unable to confirm your app's compliance. Please update your home page to allow users to view information about your app without needing to login, then retry.
- [x] Your home page does not explain the purpose of your app.
We were unable to confirm your app's compliance. Please update your home page to clearly outline the purpose of your application, then retry.
- [x] The app name "myjournal-ink" configured for your OAuth consent screen does not match the app name on your home page.
We were unable to confirm your app's compliance. Please configure your app with an accurate and up-to-date application name, then retry. Learn more.
  (updated app name is now "myjournal")

## Railway release preparation (2026-10-10)

- [x] User selected Railway at <https://myjournal-ink.up.railway.app>; retain ADR 016 residual overlap/lost-update policy.
- [x] Retain published GitHub Release trigger for stable vX.Y.Z tags; tag pushes alone do not deploy.
- [x] Prepare Docker config, pinned CLI source upload, deployment-ID polling and public health verification; see [Railway preparation](railway-preparation.md).
- [x] Railway container build/deployment and restricted-release logging review completed through public acceptance and v0.1.0. Local Docker smoke was unavailable; internal provider retention is not claimed.
- [x] HTTPS/Secure-cookie registration deployed and public OAuth checks passed. Initial 404 was superseded by successful health, sign-in and Drive smoke checks.
- [x] Fresh format/lint/application tests, seven architecture probes, both workflow actionlint checks and three deployment-polling regressions passed. Railway schema validation tooling unavailable locally; authenticated deployment remains pending.

- [x] User confirmed Railway runtime variables and exact Google callback are configured (human report).
- [x] Railway startup/public acceptance recovered from initial 502; later health, OAuth, cookie and restart checks passed.

## Railway Google startup correction (2026-10-10)

- [x] User supplied startup diagnostic confirming the old Google live-acceptance guard prevents Railway startup on current main.
- [x] Prepare non-test Google registration requiring an HTTPS origin without credentials/path/query/fragment and COOKIE_SECURE=true; retain localhost-only test mode. Regenerate Wire and add startup-policy regressions.
- [x] Fresh make generate-wire and make fmt lint test passed; lint reports 0 issues.
- [x] Acceptance build reviewed/deployed; public health, visible logging and OAuth checks passed. Release readiness subsequently allowed the successful v0.1.0 workflow.

## Public Railway HTTPS checks (2026-10-10)

- [x] Public /healthz, landing and Privacy return 200; unsigned entry API returns 401. CSP, no-store, no-referrer, nosniff and DENY framing headers observed.
- [x] Login returns 302 and ten-minute transaction cookie with Secure/HttpOnly/SameSite=Lax; invalid dummy callback returns 401; cross-origin PUT without session returns 403. No real Google callback or session values recorded.
- [x] Chrome public landing shows Continue with Google.
- [x] HSTS omission fixed and publicly verified on pages/assets/API errors/404 and after rollback restoration.
- [x] Dummy query absent from inspected application/HTTP logs; user confirmed no log drain. Real consent, cookie attributes, restart/read and synthetic entry smoke subsequently passed; undocumented internal retention is not asserted.

- [x] Human-supplied application logs show the dummy callback path without query marker; broader platform/export retention remains unverified.
- [x] Initial login 502 resolved; later browser sign-in and public acceptance passed.

- [x] Public availability recovered: fresh health GET 200 and login 302.
- [x] Initial invalid_client resolved by human configuration correction; later sign-in/consent and public Drive checks passed.

- [x] User reports corrected client ID deployed; fresh browser login reaches Google account chooser without invalid_client. Human sign-in/Drive consent pending.

- [x] Public authenticated smoke: human completed Google consent; Chrome saved disposable 2019-02-23 with Unicode text, then reload returned identical text with Saved (not Draft restored).
- [x] Public restart/read, human cookie-attribute inspection and synthetic fixture cleanup completed; evidence below distinguishes browser observations and human reports.

- [x] Railway restart acceptance: user reported service restarted with unchanged configuration; Chrome reload remained signed in and returned the exact synthetic 2019-02-23 text with Saved.
- [x] Human inspected mj_session: Secure, HttpOnly, SameSite=Lax, Path=/, expiry 2026-11-09 (approximately 30 days).
- [x] After automation timed out, user confirmed synthetic 2019-02-23 trash and reload to Blank page; cleanup complete.

- [x] Public synthetic fixture cleanup: user confirmed trash and reload of 2019-02-23 shows Blank page.

- [x] Railway public navigation smoke: calendar rendered October 2026 with 42 grid cells; entries loaded Page 1 without retry feedback. Ordinary Sign out returned the Google-login landing; subsequent protected entry navigation redirected to /.

- [x] Railway public Disconnect/reconnect (human reported): acceptance tab closed; user performed Disconnect in another Chrome tab, signed in again with the same account and explicitly granted Drive access, returning to the dated editor. Provider revocation was not independently inspected.

## Final Railway release readiness (2026-10-10)

- [x] Human confirmed Google Console remains Testing, one Railway replica, and branch autodeploys disabled.
- [x] Human reports callback HTTP logs show request IDs without query strings; export/internal retention remains a separately bounded claim.
- [x] Approved HSTS policy implemented from configured HTTPS origin, ignoring forwarded headers; one-year host-only policy, no preload/subdomains. Regression first failed against missing header, then make fmt lint test passed with zero lint issues.
- [x] Release workflow now reads RAILWAY_PROJECT_ID from secrets, matching the reported repository setup; service/environment IDs remain production-environment variables.
- [x] Deployed HSTS, rollback/session smoke, visible log/no-drain review and restricted-audience decisions completed. First published-release workflow v0.1.0 passed.

- [x] Fresh make test-architecture passed all seven forbidden-import probes after HSTS changes; git diff --check passed.

- [x] User accepted restricted test-user automated releases with Google Testing expiry and measured latency/revision-retention limitations; recorded in ADR 016.
- [x] User confirmed Railway log drains are not enabled. HTTP/application query-free observations cover configured visible logging; internal platform retention is not claimed.
- [x] Activation prerequisites completed: HSTS/workflow deployed, rollback/session smoke and restoration verified; successful v0.1.0 confirms the readiness gate allowed deployment.

- [x] Public deployed HSTS verified (2026-10-10): /healthz and landing 200, static script 200, unsigned settings API 401 and missing route 404 all include Strict-Transport-Security: max-age=31536000 and CSP. Private/non-static responses retain no-store.
- [x] Final rollback/session/restoration prerequisite completed, including independent restored health/HSTS checks.

- [x] Railway rollback/session smoke (human reported): rolled back to PR #15 deployment; health and an existing entry worked without signing in again. User then redeployed latest commit. Independent /healthz, /privacy and /help returned 200 with one-year HSTS after restoration.
- [x] Restricted test-user activation prerequisites now have recorded evidence. Repository RAILWAY_RELEASE_READY may be set to true; its remote value has not been inspected or changed. First published-release workflow remains the end-to-end automation check.
- [x] Privacy/Help Settings header regression fixed and deployed in v0.1.0. Authenticated Chrome verification shows Settings on both pages; regression tests cover invalid/expired/missing sessions retaining public access without the gear.

Privacy/Help header fix validation: fresh make fmt lint test passed with zero lint issues, all five JavaScript suites and all Go packages; git diff --check passed.

## First automated Railway release (2026-10-10)

- [x] GitHub Release v0.1.0 job 114229543859 succeeded: stable-tag/main-ancestry validation, release checks and exact deployment verification all passed. Uploaded source commit 8ae5dc70a74e3e45d59dd4da659b6efec24b8431; helper confirmed deployment SUCCESS and health.
- [x] Fresh public /healthz, /privacy and /help returned 200 with one-year HSTS and CSP. Chrome signed-in Privacy and Help both display Settings, closing the deployed header regression.
- [x] Automated release path validated for the approved restricted Google test-user audience; Console remains Testing and prior limitations remain applicable.

- [x] Release configuration follow-up: user moved RAILWAY_PROJECT_ID to a repository variable; workflow reads vars.RAILWAY_PROJECT_ID. RAILWAY_TOKEN remains a secret.

## Progress checkbox reconciliation (2026-10-10)

Reviewed every unchecked item against later findings, public acceptance and v0.1.0 workflow evidence. Closed superseded activation blockers without claiming atomic writes, controlled cold/warm benchmarks, unrestricted Google publication, local Docker smoke or full accessibility certification. Optional bounded-prefix previews are labeled deferred rather than a release blocker. The remaining unchecked umbrella is full UI acceptance as detailed above. This is a documentation audit; no new runtime tests were run.

- [x] Review README/AGENTS against current code, workflow and release evidence: document v0.1.0 restricted test-user status, HSTS, Railway variables, remaining UI checks and public-page Settings behavior; remove mandatory generation from fresh-checkout quick start. Local links and git diff --check pass; documentation-only, no new runtime tests.

- [x] README follow-up by user decision: retain make generate in quick start with prerequisites/diff review; keep evolving status in PROGRESS/findings and durable feature/setup/release instructions in README.

## Markdown formatting (2026-10-10)

- [x] Lint all 42 tracked Markdown files with markdownlint-cli2; fix formatting
  warnings and add repeatable configuration excluding private `agents/` files.
  Preserve long-line prose (MD013 disabled) and enable other default checks.
  Markdown lint and `git diff --check` pass; no runtime code changed.

## Entries loading feedback (2026-10-10)

- [x] Use one shared live status for initial HTML and pending entries requests;
  remove the duplicate list placeholder. Template and delayed-fetch regressions
  pass, along with `make fmt lint test` and `git diff --check`.

## Cache storage adapter refactor (2026-10-11)

- [x] Replace production custom map/list LRU storage with installed BigCache
  v1.2.1 under `internal/adapter/cache/bigcache`, behind the consumer-owned
  `cachestore.Backend` port. Wire constructs shared storage and supplies cleanup.
- [x] Preserve account isolation, exact snapshot expiry, Get bypass, operation
  forwarding, targeted invalidation and scalar generation guards. Cache failures
  bypass storage without changing journal mutation results; failed invalidation
  disables further cache reuse. ADR 017 records changed eviction and memory bounds.
- [x] Fresh `make generate-wire`, focused cache/adapter/provider/application tests,
  `make fmt lint test`, and `go test -race ./internal/cachestore
  ./internal/adapter/cache/bigcache` passed. All fourteen architecture probes passed separately; no temporary files remain. No live Google or production checks required
  or repeated; no performance improvement or distributed coordination claimed.

## Optional historical LRU adapter (2026-10-11)

- [x] By user follow-up, retain the previous map/list LRU storage approach under
  `internal/adapter/cache/lru`, implementing `cachestore.Backend`. Preserve the
  2048-entry default, explicit capacity override and lazy expiry; adapt storage to
  copied bytes, prefix deletion and cancelable coordination. BigCache remains
  selected in Wire; no runtime fallback or configuration option is introduced.
- [x] Fresh `make fmt lint test`, shared journal contracts through both adapters,
  `go test -race ./internal/adapter/cache/lru ./internal/cachestore`, and all fifteen
  separate architecture probes passed. `git diff --check` passed.


## Hexagonal architecture refactor (2026-10-11)

- [x] Move packages into core/driver/driven, with feature-oriented journal and
  connection cores. Keep local cache storage under driven/cache/local/{bigcache,lru}
  and rename the supporting port to cachestore.ByteCache; BigCache stays selected.
- [x] Extract connection lifecycle/settings policy, session/login models and fixed
  lifetimes inward; retain HTTP cookie/query adaptation and provider-specific OAuth
  mechanics outside core. Preserve old-cookie decoding with synthetic fixtures.
- [x] Construct renderer/catalog and narrow web options explicitly through Wire.
  Share pure date eligibility and local-origin validation; classify filesystem
  failures and stop partial-success metadata listings. Remove duplicate Clock.
- [x] Enforce recursive boundaries in .golangci.yml and prove them with thirty-one
  temporary depguard violations, including nested acyclic imports. Cleanup passed
  after both a failed probe setup and the final successful run.
- [x] Fresh make generate and make fmt lint test pass (zero lint issues, all five
  browser success markers and all Go packages). Separate make test-architecture
  passes all thirty-one probes. Race checks pass across core, driven, web and cmd/app.
  Generated output was reviewed; local documentation links and git diff --check pass.

ADR 018 records the boundaries and compatibility decisions. No new dependencies,
remote cache, live Google experiments, deployments or performance claims were
introduced. Full UI acceptance and external-writer limitations remain unchanged.
Markdown lint is unavailable on PATH; documentation paths/consistency were checked.


## Automatic import grouping (2026-10-11)

- [x] Enable the installed golangci-lint gci formatter alongside gofmt. Configure
  standard-library, third-party and project-prefix groups, alphabetically sorted
  by import path, including aliases. make fmt now formats all handwritten Go files;
  make lint rejects grouping drift. Generated files remain generator-owned.
- [x] Apply formatting repository-wide and record the convention in AGENTS.md.
  Fresh make fmt lint test passes with zero lint issues, all five browser suites
  and Go packages. golangci-lint fmt --diff ./... is empty; git diff --check passes.
- [x] Separate make test-architecture passes all thirty-one probes after formatter
  configuration, with cleanup verified.

## PR #20 review follow-up (2026-10-11)

Checked the single outstanding [review finding](https://github.com/mridha-work/myjournal-app/pull/20#discussion_r4240673309) against the current source and depguard configuration. The local AGENTS.md and ADR 017 corrections replace intermediate provider/cache paths with internal/driven/provider/{local,google} and internal/driven/cache/local/{bigcache,lru}, and identify the port as cachestore.ByteCache. The architecture table now lists both concrete local cache packages explicitly. Verified directory existence, the ByteCache declaration, the cache-storage depguard pattern, and absence of superseded names in these two guides. Documentation-only review follow-up; git diff --check passed. No remote reply, thread resolution, commit or push was performed.

## Interface assertion placement (2026-10-11)

Moved all eleven existing compile-time interface assertions immediately below their implementing struct declarations. Google Store assertions now sit together in store.go, including the OperationStore contract previously declared in discovery.go. Recorded the placement convention in AGENTS.md. Fresh make fmt lint test passed: zero lint issues, all five browser regression suites and all Go packages. git diff --check passed. Existing staged and unstaged changes were preserved; no commit or push was performed.

## PR workflow formatter prerequisite (2026-10-11)

Moved the pinned golangci-lint installation before Check formatting in .github/workflows/pr-checks.yml: make fmt now depends on that binary. The release workflow already installs it before formatting. Verified step order and fresh make fmt lint test (zero lint issues, five browser suites, all Go packages); git diff --check passed. actionlint is unavailable on PATH. No hosted Actions rerun, commit or push was performed.

## Google branding preparation (2026-10-11)

Expanded public homepage purpose/Drive disclosures and privacy policy (identity, tokens, processing, caches, browser storage, sharing, logs, retention, deletion, Limited Use, support@myjournal.ink). Refined public-page reading width, typography, wrapping and section spacing. Signed-out disclosure regressions failed before implementation and now pass. Fresh make fmt lint test passed with zero lint issues, five browser suites and all Go packages. Desktop and 390px Chrome previews inspected in dark mode; light mode checked separately. Search Console URL-prefix property was added for the Railway URL and its supplied HTML tag added to the homepage head; verification awaits deployment. See [branding checklist](google-branding-verification.md). No release, deployment, final ownership verification or branding submission performed; support mailbox delivery remains unconfirmed.

## Terms and Help pages (2026-10-11)

Added public /terms covering use, writing ownership, responsible use, availability/backups, non-atomic Drive conflicts, disconnect, service changes, privacy and support. Linked Terms from Home and Help. Help now explains saving/recovery, failures, conflicts, limited access, reconnect and trash/sign-out. Privacy, Help and Terms share the same reading-width, typography and spacing classes. Extended signed-out disclosure and session-aware Settings-link tests to /terms; observed failures before implementation, then fresh make fmt lint test passed with zero lint issues, five browser suites and all Go packages. Desktop Terms/Help and 390px Help layouts visually checked in Chrome. git diff --check passed. Terms URL added to branding checklist; no deployment, GCP configuration or submission performed.

## Project development review (2026-10-11)

Reviewed current phase requirements, implementation and release evidence at a5f761f. [Review](project-review-2026-10-11.md) separates remaining UI/branding/live reliability acceptance from optional CI, accessibility coverage and reconnect improvements. Fresh formatter/lint/vet, race-short Go tests, five JavaScript suites, three deployment tests and all 31 sequential architecture probes passed; probes cleaned up. Govulncheck reports zero reachable/imported-package vulnerabilities and one module-only advisory. PR #21 is merged; deployment/ownership/branding acceptance was not reverified. No runtime changes or renewed live Google acceptance performed.

## Native UI acceptance follow-up (2026-10-11)

Used disposable local-provider data for native Chrome 200% zoom, both-theme route reflow and keyboard traversal, conflict recovery (Keep both/Keep mine), and induced failed-save/manual recovery. Sampled custom control/focus colors exceed 3:1. Detailed observations and limits are in [UI acceptance checks](ui-acceptance-checks.md). No runtime changes or new axe scan. Full UI acceptance stays open for human screen-reader announcements, exhaustive state/focus/calendar review and external trash-icon provenance; a human screen-reader check was requested. No commit, push or deployment.

## Human UI acceptance evidence (2026-10-11)

User confirmed spoken Saved, failure and conflict feedback plus the three conflict actions. Also reported “Network data available” during failed save; its origin/order remains unverified and is retained separately. Screen reader/browser versions were not supplied. User identified SVG Repo Trash #335255; its page lists CC0, resolving the missing provenance gate (ADR 004). Basic screen-reader checks now have human-reported evidence; full acceptance still awaits the remaining exhaustive state/focus/calendar review. Documentation only; git diff --check passed. No runtime changes, commit, push or deployment.

## Focused UI acceptance completion (2026-10-11)

Completed the remaining local Chrome error/focus/calendar/confirmation checks in [UI acceptance checks](ui-acceptance-checks.md): real failed load/list/calendar, recovery, replacement Cancel/accept, trash Escape/accept, calendar marker contrast and both-theme keyboard/reflow at 390px and native 200%. Preserved the user's separate screen-reader draft and restored all temporary fixture/viewport/zoom changes. No runtime defects found. Fresh make test-js passed five suites; git diff --check passed. The focused checklist is complete using the separately recorded human announcement evidence and device waiver. This does not mark full Phase 2/3 acceptance or WCAG/cross-browser certification complete. No commit, push or deployment.

## Phase status and earlier automated checks (2026-10-11)

Added dated current-status summaries to PROGRESS, findings and phase-acceptance, superseding historical pending Google-registration and inline-theme language without removing the original evidence. Confirmed startup registration accepts non-test Google with HTTPS-origin/Secure-cookie validation and the shared layout loads external theme-init.js before CSS. Restricted Testing acceptance stays distinct from unrestricted branding/publication and unmet performance guarantees.

Expanded optional accessibility QA to 30 defined scans with signed-in Terms and an isolated signed-out Home/Help/Privacy/Terms/404 matrix, including status/no-redirect/no-Settings assertions and keyboard checks. The expanded browser/axe run was not executed here; Python syntax validation passed. Historical 18-scan evidence remains labeled.

PR checks now run JavaScript plus full race-enabled Go tests and pinned govulncheck v1.8.0, preserving the existing check name and sequential architecture probes; timeout is 30 minutes. Release checks remain. Fresh make fmt lint test passed (zero lint issues, five JS suites and all Go packages); go test -race ./... passed; govulncheck reported zero reachable/imported-package vulnerabilities and one module-only advisory. actionlint v1.7.12 passed both workflows. Local documentation links and git diff --check passed. No hosted Actions run, commit, push or deployment.

## Expanded accessibility matrix verified (2026-10-11)

Fresh owned-server synthetic Playwright/axe run passed all 30 scans with zero violations, 28 route/state keyboard checks and zero page errors (Chrome 155.0.8059.40, axe 4.10.3). Signed-in Terms and isolated signed-out Home/Help/Privacy/Terms/404 passed in both themes. Non-text arrow contrast remains axe-incomplete; independent enabled-arrow measurements passed 13.98–17.40:1. See [UI acceptance checks](ui-acceptance-checks.md) for exact command, coverage and limits. Temporary server/data cleaned up; report stays in /tmp. This supersedes the earlier preparation-only execution status. git diff --check passed. No runtime code changes, commit, push or deployment.

## Development listener and spike login limits (2026-10-12)

Resolved the two findings from the static security review. The composition root now binds local-provider and Google test-mode servers to loopback (preserving configured IPv6 loopback); production Google keeps its wildcard listener. Regenerated Wire. S01/S02 now bind IPv4 loopback and share a spike-only, mutex-protected pending-login store with a 128-transaction limit, ten-minute expiry, lazy expired-state removal and one-time consumption; full stores return 429 before OAuth redirection.

The new listener regression failed against the original wildcard behavior, then passed after the change. Fresh make fmt lint test passed with zero lint issues, all five JavaScript suites and all Go packages. go test -race ./cmd/app ./spike/... passed. Tests cover production/development listen addresses, concurrent capacity enforcement, expiry, capacity reclamation, one-time consumption, and S02 full/expired handler rejection. No live Google authorization or network-peer exploit performed; running servers must be restarted to adopt listener changes. No commit, push or deployment performed.

Sequential make test-architecture also passed all 31 forbidden-import probes with cleanup. git diff --check passed.

## PR 23 sitemap review correction (2026-10-12)

Verified the review finding that encoding/xml's flattened url>loc slice produced one URL record containing four locations. Changed serialization to four URL structs with one loc each. The strengthened regression independently decodes URL records and asserts exactly four records and one location per record; it failed against the original implementation (one record), then fresh make fmt lint test passed after the fix with zero lint issues, five JavaScript suites and all Go packages. git diff --check passed. No deployment or Search Console submission performed.

## Settings style and English/Indonesian language selection (2026-10-12)

User approved expanding the English-only interface to English and Bahasa Indonesia. Settings now shares Privacy/Help’s reading width, typography, section spacing and responsive padding, with language and account-connection sections. The language form uses a native labeled selector and an explicit save button. A session-protected, same-origin POST accepts only en/id and stores mj_language for one year (HttpOnly, SameSite=Lax, Secure on configured HTTPS). The preference is browser-specific, survives sign-out, and defaults to English when absent/unsupported. It does not alter journal text, account/session lifetime, time zone, or theme.

Both flat catalogs and their templates are loaded at startup; immutable language-specific HTTP handlers share the existing application dependencies. Server pages, JSON API error messages, browser messages, accessibility labels, and dates use the selected catalog. Added the complete Indonesian catalog and catalog-owned weekday/month names; existing browser date formatting follows the HTML language. Privacy copy now discloses language-cookie retention. Existing open pages adopt the new language on navigation/reload; changing language does not reload other tabs or interrupt their drafts.

Fresh make fmt lint test passed (zero lint issues, five JavaScript suites and all Go packages); go test -race ./internal/driver/web/... passed. Sequential make test-architecture passed all 31 probes with cleanup. Regression tests cover catalog key/placeholder completeness, Indonesian dates, default/unsupported-language fallback, selection persistence, localized API errors, signed-out rejection, HTTPS cookie protection, cross-origin/malformed/duplicate/oversized requests, and concurrent language isolation. The initial language regression failed against the original English-only implementation before the fix.

Focused local Chrome QA observed EN → ID → EN switching, Indonesian editor dates and calendar weekdays, signed-out Indonesian landing after sign-out, both Settings themes, keyboard focus, and 390px wrapping with document scroll width equal to viewport width. Screenshot evidence: /private/tmp/myjournal-settings-language-20261012.png. This is focused browser coverage, not a new screen-reader or full WCAG acceptance claim. No live Google flow, commit, push, deployment, or release performed.

### Language-picker follow-up (2026-10-12)

Moved Language after Account connection. Per user direction, the native names English and Bahasa Indonesia are literal template text, without EN/ID prefixes or locale-catalog entries. Replaced the macOS native popup with a progressively enhanced, keyboard-accessible listbox that opens eight pixels below the trigger; the underlying select remains the submitted form control and is visible when JavaScript is disabled. Added an original external chevron with padded placement, selected-state semantics, arrow/Home/End/typeahead navigation, Escape/Tab/outside-click/focus-leave dismissal, and focus restoration after selection. The form still requires explicit Save language.

Fresh make fmt lint test passed: zero lint issues, all Go packages and six JavaScript suites, including the new picker regression suite. Local Chrome confirmed reordered sections, the menu opening below the field (DOM rectangles verified an 8px gap), selection updating the trigger, and Save language applying English. Screenshot: /private/tmp/myjournal-language-picker-20261012.png. Native screen-reader behavior of the custom picker has not been human-verified. Temporary QA server/tab stopped; no commit, push or deployment.

### Settings layout refinement (2026-10-12)

Per user direction, Language is back above Account connection; its heading is Language in English and Bahasa in Indonesian. Added an original, external currentColor globe icon to the left of the selected language, preserving inset chevron spacing. Sign out and Disconnect align right. Fresh make fmt lint test passed (zero lint issues, all Go packages and six JavaScript suites). Local Chrome confirmed icon placement, section order, right-aligned actions and the Indonesian Bahasa heading after saving the language. Screenshot: /private/tmp/myjournal-settings-globe-20261012.png. No commit, push or deployment.

### Language-settings PR preparation (2026-10-12)

Generalized the Settings introduction to “Manage your preferences for myjournal.” with matching Indonesian copy. Created feat/settings-language-preferences for the user-authorized commit/push/PR. Final fresh make fmt lint test passed with zero lint issues, six JavaScript suites and all Go packages; sequential make test-architecture passed all 31 probes with cleanup. git diff --check passed. Earlier “no commit” statements describe the implementation checkpoints, not this authorized PR handoff. No production deployment or live Google acceptance repeated.

## Stable public header and signed-out Settings (2026-10-12)

User approved header order Theme → circular EN/ID language picker → Settings gear → Help. Privacy is removed from the header and remains linked from Help. Settings is public, showing language preferences and a localized sign-in prompt when no valid session exists; account data, sign-out and disconnect controls remain restricted to valid sessions. The language preference POST is now public and retains same-origin protection, bounded input, strict en/id validation and HTTPS cookie protection. Header selection submits immediately and returns only to allowlisted browser paths (including the selected entry date); external origins and API/auth endpoints cannot be redirect targets. Settings retains its explicit Save language button. No-JavaScript header language navigation goes to Settings, where the native language form remains usable.

Regression tests first failed against the previous conditional header, then fresh make fmt lint test passed (zero lint issues, six JavaScript suites and all Go packages). Focused go test -race ./internal/driver/web/... and sequential make test-architecture passed, including all 31 boundary probes. Updated expired-session tests require public Settings without account controls, while protected pages/APIs continue rejecting invalid sessions. JavaScript tests cover multiple picker initialization, header language codes, immediate submission and preservation of the entry date path.

Focused local Chrome checks confirmed public Settings, stable header before/after login, signed-out language switching, dropdown placement, 390px layout without horizontal overflow, and preservation of a selected entry date and synthetic saved text across a header language change. Screenshot: /private/tmp/myjournal-header-settings-20261012.png. The synthetic entry had been acknowledged by blur-save before language navigation; this is not evidence of an ambiguous in-flight save outcome. No live Google or native screen-reader acceptance repeated. Changes remain uncommitted; no deployment.

### Public-header accessibility regression (2026-10-12)

scripts/check-accessibility.py now expects the Settings gear in both session states, adds signed-out Settings to both themes, checks that account actions are absent there, and scans the open header language menu. Fresh local execution passed 34 axe scans with zero violations, 30 keyboard-focus checks and zero page errors. Report: /private/tmp/myjournal-header-a11y-20261012.json. Axe retains incomplete aria-controls checks and sampled contrast checks; direct DOM assertions confirmed each controlled ID is unique and references a listbox, open menus render below their trigger, and computed header-language-button contrast exceeds 4.5:1 in both themes. The existing arrow contrast assertions also passed. These checks do not establish native screen-reader announcements or full WCAG conformance. Historical “no signed-in Settings link” coverage above is superseded by the user-approved always-visible public Settings navigation.

### Header language pill (2026-10-12)

Replaced EN/ID template conditionals with {{upper .Locale}}, backed by strings.ToUpper in the renderer FuncMap. Header language navigation now uses a globe-and-code pill, including the no-JavaScript Settings-link fallback. Fresh make fmt lint test passed (zero lint issues, all Go packages and six JavaScript suites). Local Chrome confirmed the pill and no horizontal overflow at 390px. Screenshot: /private/tmp/myjournal-language-pill-20261012.png. Changes remain uncommitted.

## Footer, release metadata and date formatting (2026-10-12)

### Implemented behavior

- The footer keeps `Built for quiet writing.` independently centered on wide screens, with version and fixed 2026 copyright at the right. At 1100px and narrower, the details form a centered second line. Flex alignment centers the GitHub icon with the copyright text.
- The profile link uses a local currentColor SVG from [SVG Repo #394174](https://www.svgrepo.com/svg/394174/github), listed as CC0 when checked on 2026-10-12. The downloaded asset was inspected for SVG/path-only geometry. It has translated accessible labels, a tooltip, a 24px minimum target and visible keyboard focus. No external font, script or page request is added.
- Versions render as plain text; there is no link to the private repository’s release page. Empty versions omit both the label and its separator. Local `make run` does not read a version file or provide a VERSION override.
- The release workflow writes the validated tag to `RELEASE_VERSION` only in its temporary source archive. No such file or directory is stored in the repository, and there is no gitignore rule for it. Docker reads the optional file, validates a present value, and embeds it through `-X main.version`; an absent file yields an empty version. Wire supplies the embedded value to both locale renderers.
- Go, Docker and the release workflow use the same strict-core/flexible-suffix pattern: `vX.Y.Z`, optionally followed by dot-separated ASCII alphanumeric/hyphen identifiers. Core numbers reject leading zeros; suffix numbers may contain them. Examples include `v0.3.0-staging`, `v0.5.1-closed-beta` and `v1.2.3-beta.01`. Build metadata with `+` is not accepted. This is a project label policy, not strict SemVer validation.
- GitHub releases marked prerelease remain skipped. A regular published release with a suffix targets the existing production Railway service; a staging suffix does not select another environment. Readiness, main ancestry, code checks and deployment verification remain required.
- The composition-root logger attaches nonempty `main.version` using `slog.Logger.With`. Request logs, derived loggers and application failures inherit it; main installs an info-level logger before configuration loading, then replaces it with the configured logger. Empty local versions omit the field.
- Date formatting reuses translated `date.weekday.*` and `date.month.*` names. English uses ordinal days (`Thursday, October 1st 2026`); Indonesian uses a zero-padded day (`Kamis, 01 Oktober 2026`). The old `format.date` locale key is removed. Other locales still use the English Go-layout fallback; a shared formatting group or locale-owned pattern was discussed but not implemented.

### Recorded evidence and remaining checks

- Implementation checkpoints recorded passing `make fmt lint test` (zero lint issues, six JavaScript suites and all Go packages). Tests cover both locales, version labels/separator omission, invalid versions and version inheritance in logs. Workflow Bash and Docker shell checks covered stable/suffixed/malformed/empty/missing metadata, and confirmed identical regex patterns. These are recorded implementation results, not newly rerun checks for this documentation edit.
- Earlier footer QA used a synthetic `v1.2.3` binary: Chrome confirmed exact desktop tagline centering at 1512px and a two-line 390px layout without horizontal overflow. Screenshot: `/private/tmp/myjournal-footer-desktop-20261012.png`. That check preceded the final icon/alignment and version-link changes; it does not establish the final visual result or a deployed release version.
- Three offline deployment lifecycle tests and the sequential 31-probe architecture check passed during the initial footer implementation. The temporary QA server, binary and browser tab were cleaned up.
- Final footer visual/browser and native screen-reader checks have not been repeated. Docker daemon access was unavailable during initial QA; no Docker image build, production deployment or live release-version log check has been performed for this work.

These consolidated notes supersede the intermediate footer/version experiments from this session. Documentation reconciliation checked current code and repository state; `git diff --check` passed. No runtime tests were rerun for documentation-only edits.

Commit preparation (2026-10-12): fresh make fmt lint test passed with zero lint issues; three offline deployment-helper tests and sequential make test-architecture (31 probes, cleanup complete) passed. git diff --check passed. User authorized committing the current working-tree changes on chore/adding-version-tag-and-ui-improvement. No push, deployment, Docker image build or new browser acceptance was performed.

## Bounded overlapping journal reads (2026-10-12)

Implemented provider-independent request collapsing in the journal cache decorator using `golang.org/x/sync/singleflight` ([ADR 019](adr/019-overlapping-journal-reads.md)). It shares index misses, editor reads, identical preview batches and matching dates across different batches. The composition root supplies the validated provider/account, sharing both active reads and TTL snapshots across devices with different grants. Wire was regenerated. The earlier uncommitted Google-specific coordinator was replaced at the user's request.

Separate bounded batch/read coordinators prevent nested admission deadlocks; each admits 64 distinct keys and 320 registrations, including cancelled callers until their result drains. Leaders use request-derived 30-second deadlines. Surviving followers retry cancelled leaders through their own Store/OAuth context. Completed editor content/errors are discarded. Save/trash advance generations and remain independent; existing TTL invalidation/generation guards remain intact. Google keeps operation-scoped discovery reuse, using singleflight with short metadata locks and generation-safe clearing. `.golangci.yml` narrowly permits singleflight/errgroup in cache policy without allowing concrete providers or storage adapters there.

Controlled eight-request fixtures recorded Entries 32 lists/160 downloads versus 8/20, Calendar 24/0 versus 3/0, mixed browsing 28/80 versus 8/20, and editor reads 24/8 versus 3/1. Partially overlapping preview batches retain the four-to-three download reduction. See [measurements and limitations](performance-benchmark.md#overlapping-drive-reads-2026-10-12). These are mocked request counts, not live Drive latency. The earlier Google-specific 4-list Entries/mixed results are superseded by this provider-independent implementation.

Fresh validation: `make fmt lint test` passed (zero lint issues, all Go packages and six JavaScript suites); focused race checks covered Google, journal cache and composition-root packages. Concurrent browsing, OAuth cancellation, post-mutation reads and overlapping previews passed twenty repetitions. Sequential architecture validation and final checks are recorded below. No live Google traffic, deployment, commit or push was performed. Cross-process coordination, active cancellation on disconnect and atomic external-writer protection remain outside this change.

Final verification for the shared implementation: `make fmt lint test` passed with zero lint issues; `go test -race ./internal/driven/cache/journal ./internal/driven/provider/google ./cmd/app` passed. The sequential `make test-architecture` passed all 31 forbidden-import probes with cleanup complete. `git diff --check` passed. Discovery-clear regression coverage confirms that older active loads neither attract post-clear callers nor refill the cleared snapshot. No commit or push was made.

Account-level sharing refinement: the user explicitly selected provider/account as the shared-read boundary across laptops/grants. Removed the credential digest and `WrapScoped`; the composition root again calls `cache.Wrap(providerID+":"+accountID, store)` after provider validation, and Wire was regenerated. Successful reads share across request-owned Stores. A follower of `ErrAuthRequired` performs one bounded retry through its own Store, retaining admission until that retry ends; leaders and failed independent retries return their own errors. Regression tests first failed without authentication retry, then passed for shared success, expired-leader/valid-follower and both-expired cases. Fresh `make fmt lint test` and focused Google/cache/composition-root race checks passed. Existing cancellation and mutation-generation behavior is retained.

Account-sharing final check: sequential `make test-architecture` passed all 31 probes and removed its temporary files; `git diff --check` passed. Changes remain uncommitted.

## Recent Entries and handler service contract (2026-10-12)

Renamed core `Service.Page` to `ListEntries` across callers/tests/benchmarks. Confirmed the existing loop stops at the requested page plus one date; a new twenty-entry regression verifies that it skips older years when the current year supplies the lookahead. Visited years still load complete metadata indexes, and lookahead may visit an older year when exactly twenty dates exhaust the newest year.

By user decision, browser Entries now always requests `/api/v1/entries?page=1&size=20`, ignores pagination/size URL parameters, removes Older/Newer controls and describes the latest-twenty limit with a Calendar link in both locales. Retry stays on the first page; malformed oversized responses are rejected. API pagination/bounds and the JSON envelope are retained. Updated README, acceptance and browsing checklists.

`handler.JournalService` is the consumer-owned HTTP use-case contract; `ServiceFactory` returns it while core constructors stay concrete. Regenerated Wire. A stub-service handler regression verifies successful/error responses and forwarded arguments without constructing the real journal service. [Concurrency/interface triage](go-concurrency-interface-review.md) records current parallel preview behavior, dependent operations and measured-follow-up candidates; no speculative year prefetch or blanket interface conversion was added.

Fresh `make fmt lint test` passed (zero lint issues, all Go packages and six JavaScript suites). Focused race checks passed for journal, web, cache, Google and composition-root packages. Chrome with 25 temporary entries confirmed the latest twenty, fixed page/size, absent pagination, older Calendar access, Indonesian copy and no horizontal overflow at 390px. Accessibility initially caught an inline-link distinction failure; a persistent underline fixed it, and a fresh rerun passed all 34 axe scans and keyboard checks. The report retains incomplete checks and does not establish full WCAG conformance or new screen-reader evidence. Owned QA servers/data were cleaned up. No live Drive traffic, commit, push or deployment was performed.

Recent Entries final verification: sequential `make test-architecture` passed all 31 forbidden-import probes and cleaned its temporary files; `git diff --check` passed. Changes remain uncommitted in the current branch.

## Cold Entries follow-up (2026-10-12)

The inline Open calendar link and successful-load announcement were removed by user request. The description still mentions Calendar, available through header navigation; loading, errors and Retry retain accessible feedback.

Google discovery now derives year-specific folder matches from a complete broad year-folder result already retained within the same operation. Missing or oversized uncached results still use the original query. This removes redundant metadata requests without retaining discovery across operations or changing mutation checks. Cache-decorated previews now permit eight concurrent downloads; direct Google batches retain four workers and avoid idle singleton workers.

Controlled synthetic measurements (not live Drive timings):

| Workload | Metadata calls | Content downloads |
| --- | ---: | ---: |
| Cold eight-entry, two-year list | 4 (previously 6) | 8 |
| Immediate warm list | 0 | 0 |
| List after updating one visible entry | 3 | 1 |

With a synthetic 100 ms download delay, eight previews dropped from two waves/200 ms to one wave/100 ms; twenty dropped from five waves/500 ms to three/300 ms. Peak concurrency was eight. These results do not establish live Drive speed or throttling behavior. The reported seven-second cold/post-save latency remains unmeasured against Google after this change. Post-save discovery and the invalidated preview remain external work; progressive previews would require a separate API/UI design.

Fresh validation passed: `make fmt lint test`, focused cache/Google/web race checks, all 31 architecture probes, and 34 Chrome axe scans plus keyboard checks. Accessibility automation retains its documented coverage limits. No live Drive measurement, deployment, commit or push was performed.

## Entries limit and preview concurrency (2026-10-12)

Cache preview concurrency is now ten by user decision. Synthetic 100 ms downloads take one wave for eight previews and two for twenty; this is not live Drive latency evidence. The Google direct-batch limit remains four.

`internal/driver/web/render.go` owns `recentEntriesLimit` (currently 10). Change this constant within the API-supported range 1–100 to change browser requests, response-size validation and both localized descriptions together. Templates expose it through `data-list-limit`; locale strings use `{listLimit}`. API pagination limits remain unchanged. A non-default five-entry JavaScript regression checks the configurable request/response path, and renderer tests check both locales.

Fresh `make fmt lint test`, cache race checks and `git diff --check` passed. Changes are uncommitted; no deployment or live Drive measurement was performed.

The browser limit was subsequently reduced to 10 by the user. Active README, acceptance/browsing guidance and JavaScript fixtures now reflect ten; earlier twenty-entry QA and benchmark records remain historical evidence. Production JavaScript and localized descriptions already derive the limit from the rendered page and needed no adjustment.

Pre-PR review found no blocking issue. Fresh `make fmt lint test`, `go vet ./...`, `go test -race -short ./...`, all 31 architecture probes and `git diff --check` passed. The reviewed branch includes bounded shared reads, operation-local discovery reuse, ten preview workers and the configurable ten-entry browser view. Live Drive speed/throttling and cross-process guarantees remain unverified.
