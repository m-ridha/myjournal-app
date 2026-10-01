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
- [ ] Complete the live Google Cloud Console consent and OAuth sign-in flow with a test account
- [ ] Verify refresh-token behavior and first-drive read against the real account
- [ ] Record final findings in `docs/findings.md`

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

- Human module path confirmation is pending; the repo keeps the existing module declaration from `go.mod`.
- Live Google sign-in still depends on Google Cloud Console setup and a real consent flow.
- Full Google Drive provider behavior remains human-assisted until the real OAuth and Drive validation steps are run.
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
