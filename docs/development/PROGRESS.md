# Progress

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
- [x] Google provider selection is rejected until its OAuth and Drive implementations are complete
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
- [ ] Remaining Phase 2/3 security/API/UI acceptance gates remain open

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

## Blockers

- Preserve the user-selected module declaration from `go.mod`; module renaming is not authorized.
- Live Google consent and the S1 first-consent/refresh/claim checks are complete for the test account.
- Full Google Drive provider behavior remains human-assisted until the real OAuth and Drive validation steps are run.
- S2 folder behavior checks passed for the test account (human-reported 2026-10-08); production Google storage remains unimplemented.
- Phase 1 evidence/fallbacks and the user go decision are recorded, including S3 single-process concurrency. Phase 4 remains gated on Phase 2/3 acceptance and validated Google implementation; cross-instance folder convergence, revision/latency measurements, and production OAuth checks remain explicit follow-ups.
- Session age, unsafe-request cross-origin protection, save-envelope bounds, and query-free request logging are implemented. HTTP timeout/draining and CSP work, editor auto-save/page-hide, and Google OAuth state/PKCE persistence remain incomplete. Reported browser time-zone checks passed; physical-phone testing is waived for this stage.

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
