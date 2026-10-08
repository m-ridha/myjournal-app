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
- [x] Entry list and preview generation are implemented
- [x] Preview logic trims leading blank lines and uses the first meaningful line
- [x] 404 routes render the branded not-found template rather than raw Go output

## Phase 3: UI and polish

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
- [x] `internal/web/middleware` scaffolded as an empty extension point
- [ ] Middleware behavior remains deferred until the relevant development phases

## Blockers

- Preserve the user-selected module declaration from `go.mod`; module renaming is not authorized.
- Live Google consent and the S1 first-consent/refresh/claim checks are complete for the test account.
- Full Google Drive provider behavior remains human-assisted until the real OAuth and Drive validation steps are run.
- S2 folder behavior checks passed for the test account (human-reported 2026-10-08); production Google storage remains unimplemented.
- Phase 1 evidence/fallbacks and the user go decision are recorded, including S3 single-process concurrency. Phase 4 remains gated on Phase 2/3 acceptance and validated Google implementation; cross-instance folder convergence, revision/latency measurements, and production OAuth checks remain explicit follow-ups.
- Session expiration, CSRF, browser-timezone handling, editor auto-save/page-hide, and Google OAuth state/PKCE persistence remain incomplete.

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
