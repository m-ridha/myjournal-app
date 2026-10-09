# Findings Log

## Favicon (2026-10-08)

- Cleanup: the current favicon is `/static/favicons/myjournal-tab-icon.png`, embedded from `assets/static/favicons/`. Updated the shared layout and removed unused SVG/mask assets at the user's request; paths below describe historical troubleshooting. `make fmt lint test` passed (lint: 0 issues; browser-script regressions and all Go tests passed), and `git diff --check` passed. Verified only the active PNG remains among favicon assets and the layout references its new path.
- Resolution (human-reported): after quitting Safari and renaming `~/Library/Safari/Favicon Cache` to retain a backup, the user confirmed the app favicon displays at `http://localhost:3030/` in Safari 26.4. This identifies persisted Safari icon-cache state as a contributing factor; the earlier asset changes alone did not resolve it. The current layout uses only `/static/favicons/myjournal-tab-icon.png`, with no version query. The earlier pending Safari normal-tab checks below are superseded by this confirmation; pinned-tab behavior and SVG dark-mode adaptation after the reset remain unverified. The temporary port-3031 diagnostic/proxy server was stopped. Documentation validation: `git diff --check` passed.
- Isolation test: the user reported that neither normal nor pinned Safari tabs recovered with the PNG/SVG/mask combination, including at `127.0.0.1:3030`. A temporary loopback server on port 3031 served the same PNG with a single `rel="icon"` link and no app scripts/cookies. The user confirmed Safari displayed its tab icon, and server logs confirmed requests for `/journal-test.png` returned 200. Switched the app to a single PNG link at a fresh unversioned filename, `/static/favicons/myjournal-tab-icon.png`; retained the SVG source and unused mask asset. This tests icon selection/cache differences, without proving which caused the failure. The PNG uses a fixed light palette rather than SVG dark-mode adaptation. App normal/pinned Safari and Chrome confirmation remains pending.
- Safari PNG fallback did not resolve the user's normal/private tab failure. Reviewed the user-linked [Stack Overflow answer](https://stackoverflow.com/a/68925887) and [Apple's pinned-tab guidance](https://developer.apple.com/library/archive/documentation/AppleApplications/Reference/SafariWebContent/pinnedTabs/pinnedTabs.html): `mask-icon` is documented for pinned tabs, not a proven normal-tab fix. Added a separate black, transparent, single-layer 16×16 notebook/pencil silhouette and `mask-icon` link; refreshed all icon URLs with `?v=3` for cache troubleshooting. Normal and pinned Safari tab behavior remains unverified.
- The user confirmed Chrome works without the version query after clearing its cache. Safari 26.4 (21624.1.16.11.4) still showed a letter fallback on the app tab in private browsing despite rendering the SVG directly. Added `/static/myjournal-favicon.png` (64×64 RGBA) before the SVG link as a fallback. Rasterized a temporary copy with explicit light-palette fill/stroke attributes using macOS AppKit; visually verified the notebook/pencil and transparency. The SVG remains unchanged. `make fmt lint test` passed; after correcting the raster output, `go test ./internal/web/...` and `git diff --check` passed. Safari recovery is pending human confirmation; no exact Safari failure cause is established.
- Chrome troubleshooting: the user reported a missing tab icon despite restart/private browsing and a working direct SVG URL. Live HTTP checks confirmed `200 OK`, `image/svg+xml`, and the correct link in the landing HTML. Added explicit 64×64 SVG dimensions and a fresh favicon URL (`?v=2`) as a compatibility/cache workaround. The user confirmed the tab icon works after this change (2026-10-08); the individual contribution of dimensions versus cache refresh is not established.
- The user-provided SVG is served at `/static/myjournal-favicon.svg` by the existing embedded static handler and linked from the shared layout. Its original `prefers-color-scheme` styling is preserved; favicon colors follow the browser/system preference.
- Validation: `make fmt lint test` passed (lint: 0 issues; editor regressions and `go test ./...` passed); `git diff --check` and an exact source/asset comparison passed. Browser tab appearance remains manually unverified. The formatting target also formatted four Go files with existing workspace changes.

## Bootstrap

- `go.mod` was already present and has been preserved for this repository.
- A minimal Go bootstrap was created to satisfy the Phase 0 acceptance criteria.
- `make fmt lint test` passed successfully in the current workspace after installing the correct golangci-lint v2 toolchain and aligning the config to the guide’s architecture.
- Google OAuth environment values are now present in the local project environment for the next Phase 1 work.

## Verified local app behavior

- The app routes are versioned under `/api/v1` and the entry pages use the expected date-based paths.
- Future dates are rejected by both the router and the API layer.
- A missing or invalid route renders the branded not-found page instead of the raw default Go `404 page not found` response.
- The entry preview logic now trims leading blank space and uses the first meaningful text line as the preview, without counting the entire entry length as the preview length.
- The editor shows a visible save-state cycle: saving → saved → neutral while dirty again.
- The UI text was shortened from `Saved locally` to `Saved` to keep the status cleaner and less noisy.
- Client-side draft persistence keeps text locally when a save fails or the page is left mid-edit.

## Phase 1: Google OAuth spike

- The provider seam and Google config scaffolding are in place.
- A throwaway PKCE spike is available in `spike/s01_google_oauth` and is designed to exercise the live login and initial Drive API read path.
- Human-reported S1 result (2026-10-01): forced-consent callback succeeded; Drive list succeeded with zero visible files; refresh token was present; simulated-expiry refresh succeeded; Drive list using the refreshed token succeeded; ID token was present and validated for the configured client ID; `sub` and `email` claims were present.
- This verifies the S1 first-consent, refresh, claim-validation, and initial Drive-call acceptance checks for the test account. The zero file count is a successful empty listing, not a Drive API failure.
- The S01 callback reports only booleans and a count. It never prints token values, claim values, or file names. Renewal is tested by expiring a copy of the access token, then making a Drive call using the renewed token.
- Prior human-reported S1 comparison, preserved in AGENTS.md: a subsequent sign-in without forced consent returned no refresh token; ID-token validation still succeeded. Production OAuth must preserve an existing refresh token when a later response omits it.
- S4 conflict-field selection and S7 cookie/token lifecycle verification are complete for the test account, with the documented Safari HTTP localhost workaround (see live results below). Phase 2/3 acceptance still gates Phase 4. Remaining nonblocking experiments need evidence or recorded fallbacks, followed by the human go/no-go review.
- Human-reported S2 result (2026-10-01): creating the uniquely tagged root/entries/year folders and initial metadata lookup succeeded. After renaming the root, verification found the same folder IDs and returned the renamed root name. This confirms lookup survives a root rename for this test account.
- Historical S2 failure: the first `check-trash` run reported an active entries folder and did not find the original entries ID in the trashed query. A premature `recreate-entries` run returned two active entries folders; the earlier harness may have left an extra test folder/year pair. The updated harness refuses recreation while an active entries folder remains and buffers failed output. The successful rerun below supersedes the earlier failed trash check.
- Human-reported S2 preparation (2026-10-08): completed inspection of the existing test folders and preparation of a manually created same-name root folder and text file in the original year folder.
- Human-reported S2 visibility result (2026-10-08): `visible_root_matches_app_created_root_only: true` and `manual_file_invisible_to_app: true`. The app query saw only the app-created root and did not see the manually added text file for this test account.
- Human-reported S2 trash result (2026-10-08): `entries_absent_from_active_query: true` and `entries_found_in_trashed_query: true`. After the original entries folder was moved to trash, it was absent from the active query and its original ID was found in the trashed query.
- Human-reported S2 recreation result (2026-10-08): `metadata_lookup_ids_match: true`, `entries_recreated_with_new_id: true`, and `year_recreated_with_new_id: true`. Guarded recreation created replacement entries/year folders under the existing root and verified their metadata lookup. Replacement resource IDs remain local rather than being copied into this log.
- Human-reported S2 move result (2026-10-08): `moved_year_folder_found_by_properties: true`. The moved year folder remained discoverable by its app properties; this does not establish that it still belongs to the intended hierarchy or that it was moved back afterward.
- S2 folder lookup, rename, visibility, trash detection, guarded recreation, and moved-year checks have passed for the test account. These results validate the spike operations; the production Google Store must still implement and validate read-without-creation and recreate-on-write behavior.

## S02 review fixes (2026-10-08)

- Visibility mismatches now include both diagnostic booleans in the returned error, preserving them in the callback's existing local failure log even when buffered browser output is discarded. The additional diagnostics contain no filenames, resource IDs, or credentials.
- Query values escape backslashes before apostrophes, following [Google's Drive query guidance](https://developers.google.com/workspace/drive/api/guides/search-files). Regression checks cover mixed escapes and mocked visibility success/root/file/both mismatch results.
- `make fmt lint test` passed: lint reported 0 issues, editor interaction regressions passed, and `go test ./...` passed including the new S02 checks. No live Google experiment was rerun.

## S04 harness preparation (2026-10-08)

- Added the separate `spike/s04_drive_conflicts` harness, preserving S02 as the completed folder experiment. S04 creates one tagged synthetic text file, compares `version`, `modifiedTime`, `md5Checksum`, and `headRevisionId` after a content edit and API rename, and records one metadata-read timing sample. A local metadata-only baseline supports a subsequent read-only check after a manual Drive rename.
- Mocked HTTP checks exercise uploads, candidate availability/comparison, refusal to upload a stale client's content, preservation of the newer checksum, read-only manual verification, and refusal to overwrite the baseline file. These checks validate the harness, not live Google behavior.
- `make fmt lint test` passed: formatting completed, lint reported 0 issues, and all Go tests passed. The initial sandboxed lint attempt failed to load Go packages; the checks succeeded outside the sandbox after correcting the mock parser to handle multipart/related uploads.
- At harness preparation, live S4 results were pending; the subsequent human-reported results are recorded below. A pre-save metadata comparison does not prove atomic conditional updates or close the check/update race; checksums do not distinguish an A→B→A content history. S7 remains a separate next experiment.

## S4 live results and decision (human-reported 2026-10-08)

- All four candidate fields were available and changed after the synthetic content edit. Each candidate rejected the stale client A snapshot in the harness's pre-save comparison, and `client_b_content_preserved: true` confirmed the newer checksum remained current. The stale client did not upload its content.
- Both the API rename and the human-confirmed rename in Drive changed `version` and `modifiedTime`, while `md5Checksum` and `headRevisionId` remained unchanged and available.

| Candidate | Changed on content edit | Changed on API rename | Changed on manual rename |
| --- | --- | --- | --- |
| `version` | Yes | Yes | Yes |
| `modifiedTime` | Yes | Yes | Yes |
| `md5Checksum` | Yes | No | No |
| `headRevisionId` | Yes | No | No |

- Decision: use Drive `md5Checksum` as the production Google adapter's opaque content version token for plain-text entries; ADR 005 records the choice. `headRevisionId` also passed the observed comparisons, but the checksum directly represents the content being protected. `version` and `modifiedTime` would produce conflicts on the observed renames. The core's `journal.Version` stays provider-independent.
- The pre-save metadata read required one request and took 349 ms in this run. This is a single sample, not a percentile or performance guarantee.
- S4 field selection is complete for this test account and these operations. One content edit and two rename checks do not establish behavior for every account, sharing changes, or concurrent writes. Production Google storage is still unimplemented; atomic conflict protection and the check/update race remain unresolved. No resource IDs, filenames, or content were copied into this evidence log.

## S7 harness preparation (2026-10-08)

- Added `spike/s07_cookie_lifecycle` with distinct spike cookies, reusing `internal/session` AES-GCM encoding without changing production auth. The encrypted session carries provider/account identity, email label, refresh token, and an existing S02 root folder ID. Drive checks read root metadata only; no folders or entries are created, edited, or revoked automatically.
- OAuth state/PKCE verifier use a five-minute encrypted HttpOnly cookie. The callback validates the ID token and verified email; a missing refresh token on a later sign-in can reuse the existing token only for the same validated subject. Tokens, claims, cookies, callback queries, and raw external errors are suppressed from output.
- The access-token cache is bounded to 128 items and keyed by a SHA-256 refresh-token hash. It stores token values rather than request-context-bound token sources; an empty cache refreshes from cookie credentials. Forced renewal simulates expired access credentials. `invalid_grant` clears the session and redirects to the sign-in page; transient failures retain the cookie. Returned replacement refresh tokens are persisted without extending the session deadline.
- The spike proposes a fixed 30-day lifetime, with server-side `IssuedAt` enforcement, and checks the full Set-Cookie representation against a 3800-byte budget. These are experiment settings; actual Google-token size, browser storage behavior, and the final production lifetime remain pending.
- Mocked checks cover cold/warm cache, expired access token, a fresh server instance using the same browser cookie, forced refresh after revocation, cookie deletion/redirect, tampering, key rotation, cookie attributes/size, age boundaries, encrypted pending login, state mismatch, and cross-origin POST rejection. They do not establish live Google or browser behavior.
- `make fmt lint test` passed: lint reported 0 issues and all Go tests passed. `spike/s07_cookie_lifecycle/README.md` provides the human-assisted sequence, including manual Google revocation and Safari/Secure-on-localhost observations. S7 remains open until that evidence and the lifetime decision are recorded; production Google registration remains blocked.

## S7 initial live attempt (human-reported 2026-10-08)

- Forced-consent sign-in reached the Drive root check, which failed with the original generic error. No new session cookie was issued; the subsequent `/check` redirected with `result: session_rejected`. This does not establish a cookie encryption or refresh failure. The underlying root/API error is still unknown.
- Added sanitized Drive failure diagnostics: an allowlisted category, Google HTTP status when available, and whether a session was retained. Trashed roots and non-folder IDs are distinguished. Raw API messages and resource IDs remain suppressed. The user must rerun sign-in with the existing S02 root ID and report the safe diagnostics before continuing lifecycle checks.

## S7 live lifecycle results (human-reported 2026-10-08)

- A subsequent forced-consent sign-in succeeded: the ID token validated, the existing root metadata read succeeded, and the session cookie was issued. The earlier 404 root-read failure did not recur; its exact cause was not established.
- The callback reported a 584-byte Set-Cookie representation, below the 3800-byte budget, with HttpOnly, SameSite=Lax, Secure=false, and a proposed 30-day lifetime. These were initially server-reported attributes; subsequent browser inspection is recorded below.
- The first `/check` accepted the cookie, refreshed on a cache miss, and read Drive successfully. The second reused the cached access token. `/check?force_refresh=1` bypassed the cache, simulated expiry, refreshed, and read Drive successfully.
- Restart recovery passed in the original browser: the first confirmed check after restarting reported a cache miss, successful refresh, and successful Drive read without signing in again. An earlier reported cache-hit result did not prove a cold restart. A separate private browser rejected the session as expected because it did not share the original browser cookie.
- The tampering action redirected with `result: session_rejected`; re-sign-in and refresh then succeeded. Explicit session clearing redirected with `result: signed_out`. Changing the encryption key rejected the prior cookie; restoring the original key allowed a fresh forced-consent sign-in. Browser-side deletion of the cookies was not independently inspected.
- A subsequent `/login` without forced consent reported `refresh_token_returned: false` and `existing_refresh_token_preserved: true`. The following `/check` refreshed from that preserved token and read Drive successfully.
- After the human removed this app's third-party connection in Google Account settings, forced renewal redirected with `result: invalid_grant`. The harness clears the session on this path and presents sign-in links; the human observed the redirect/result, not a separate cookie-storage inspection.
- Live lifecycle checks passed for the test account. Subsequent browser observations, the lifetime decision, and the Safari HTTP localhost workaround are recorded below; production expiration, rotation, and Google provider implementation are unchanged.

## S7 browser observations and lifetime decision (2026-10-08)

- Human inspection in Google Chrome 154.0.8037.98 (Official Build, arm64; human-reported) confirmed `mj_s7_session` was absent after revoked-access handling. After reauthorization, Chrome stored the spike cookie with HttpOnly checked, SameSite=Lax, Secure off, path `/`, and expiry on 2026-11-07, consistent with the 30-day lifetime. Secure-on-localhost/Safari observations are recorded below.
- The user adopted a fixed 30-day session lifetime for now, with the option to revise it to one or two weeks during development. ADR 002 records the decision. Production server-side age enforcement is still pending; the spike already enforces its original `IssuedAt` deadline.
- A session cookie value was shared during inspection; it is intentionally omitted from documentation. The user was advised to revoke the app's Google connection and force refresh to invalidate the embedded credentials and clear the spike cookie. Completion of that remediation has not been reported.

## S7 Secure-on-localhost comparison (human-reported 2026-10-08)

- With `COOKIE_SECURE=true`, Google Chrome 154.0.8037.98 completed sign-in, reported a 592-byte Set-Cookie representation, and accepted the session on `/check`; the cache refreshed and the Drive root read succeeded. The human confirmed Secure was checked on `mj_s7_session` in browser storage.
- Safari Version 26.4 (21624.1.16.11.4) returned `OAuth state rejected` with Secure enabled on HTTP localhost. This is an OAuth-state-stage failure, not proof of session decryption or token-refresh failure. The harness requires the temporary `mj_s7_oauth` cookie at that stage; failure to store/send it is a possible explanation, not yet confirmed by cookie inspection or a Secure-disabled comparison.
- The reported Safari login URL used `foce_consent` instead of `force_consent`. That omitted forced consent, but does not disable the harness's state cookie/check and does not by itself explain the state rejection. The subsequent Secure-disabled comparison using the correct parameter is recorded below. Production HTTPS behavior remains unverified.

## S7 Safari workaround and acceptance (human-reported 2026-10-08)

- With `COOKIE_SECURE=false` and the correctly spelled forced-consent URL, Safari 26.4 (21624.1.16.11.4) completed sign-in, validated the ID token, read the root, and issued the cookie. The callback reported a 584-byte Set-Cookie representation, HttpOnly, SameSite=Lax, Secure=false, and a 30-day lifetime. `/check` accepted the cookie, refreshed on a cold cache, and read Drive successfully. Browser inspection confirmed the cookie was stored for localhost with path `/`, SameSite=Lax, and a November 7 expiry; its 495-byte browser size is a different measurement from the full Set-Cookie header.
- Decision: use `COOKIE_SECURE=false` for plain-HTTP localhost development. Chrome accepted Secure cookies there, while Safari's Secure-enabled attempt failed at the OAuth-state stage and its Secure-disabled attempt succeeded. The temporary-cookie storage/send explanation remains an inference; the successful fallback is observed. Keep Secure cookies required for production HTTPS; that environment was not tested by this spike.
- S7 experiment acceptance is complete for the test account with Chrome/Safari observations, the localhost fallback, mocked lifetime boundary checks, and the user-approved fixed 30-day lifetime. This does not establish every lifecycle case in both browsers or production Google readiness. S1/S2/S4/S7 blocker evidence is now recorded; remaining nonblocking spike evidence/fallbacks, Phase 2/3 acceptance, and the human go/no-go review are still required.

## Phase 1 continuation after user go (2026-10-08)

- User gave a go decision to continue after the blocking spike evidence. `docs/development/phase-1-review.md` records the evidence/gates, S5/S6 conservative fallback plans, S11 official-doc research, and S12 existing-library choices. These are explicitly separated from live measurements and production acceptance.
- Added `spike/s03_folder_concurrency`: five concurrent naive ensure-path calls, then five calls serialized by a shared in-process lock, each using fresh uniquely tagged test folders. The harness counts each role and checks worker path agreement. It selects the oldest query result deterministically, preserves duplicates, and never cleans up Drive items automatically. Live results and multi-instance/eventual-consistency behavior remain pending.
- Deterministic fake-storage tests force the naive root-creation race and verify locked path reuse and stable oldest-wins selection without deleting duplicates. `make fmt lint test` passed with 0 lint issues and all Go tests passing. This validates the harness, not Google consistency.
- `go test -race ./spike/s03_folder_concurrency` passed. `git diff --check` also passed after the documentation updates.
- Research finding: the documented Testing refresh-token limit applies when requesting Drive access; the approved 30-day session lifetime cannot override it. Production publishing/brand/domain work remains deferred to hosting and release acceptance. Sources and precise limitations are linked in the review document; no Console changes were made.

## S3 live concurrency results (human-reported 2026-10-08)

- Five simultaneous naive list-then-create workers produced five root folders, five entries folders, and five year folders. `naive_workers_same_path: false` demonstrated the path-creation race for this test account.
- With a shared in-process lock around the entire ensure-path sequence, the same worker count produced one folder per role. `locked_workers_same_path: true`, `locked_one_folder_per_role: true`, and `S3_SINGLE_PROCESS_CONCURRENCY: passed` confirmed that all workers used the same path.
- Decision: production ensure-path operations need a shared per-provider/account lock across session-scoped stores, plus deterministic oldest-wins handling for existing duplicates. Preserve duplicate/data-bearing folders for inspection; this run did not test automatic merging, cross-instance locking, or eventual list consistency. No test folders were cleaned up automatically.
- Phase 1 evidence/fallbacks and the user's go decision are now recorded in `docs/development/phase-1-review.md`. S5/S6 are conservative documented fallback plans, not live revision/latency acceptance. Production OAuth setup, Phase 2/3 acceptance gaps, and validated Google implementation remain outstanding.

## Editor text safety follow-up (2026-10-08)

- The editor restores account-scoped drafts over server text and keeps recovered text dirty against the server baseline. Drafts carry the version they were based on, so recovery of stale text does not silently adopt the newest server version. Empty and whitespace-only edits are retained; loading cannot overwrite text typed while the GET is pending. Save stays disabled until the server baseline is known, with a localized retry-loading action after failures.
- Save acknowledges only the submitted snapshot. Text typed during a PUT stays dirty and backed up against the newly returned version; Save remains disabled until the in-flight request settles. Conflicts, expired sessions, network failures, and invalid response bodies preserve text and restore usable controls. Storage errors are guarded and do not falsely claim that a backup was kept.
- Reverting to the previously loaded text while a different save is in flight still keeps a backup and leave warning. If the response is lost, the editor treats the outcome as uncertain rather than clearing that text as clean; a regression covers this case.
- New draft keys use an opaque SHA-256 scope over provider/account identity plus the selected date. Labels and credentials are not exposed in that scope. Older date-only drafts are never automatically assigned to an account: an explicit confirmation offers recovery, and migration removes the old copy only after a successful scoped write.
- Settings Sign out now clears the current account's drafts and legacy unscoped drafts, preserves unrelated account/theme keys, and broadcasts a storage marker to stop other open editor tabs from recreating drafts. Failed cleanup is visible before the logout request; failed logout is retryable. Disconnect remains separate and unwired.
- Dependency-free JavaScript interaction regressions run through `make test-js`, using Node.js or macOS JavaScriptCore, and are included in `make test`. Go tests cover provider/account scope separation, stable scope across label changes, and rendering without credentials. `make fmt lint test` passed: 0 lint issues, editor interaction regressions passed, and all Go tests passed.
- Manual browser checks are documented in `docs/development/editor-recovery-checks.md`; none were reported or automated in a real browser during this change. Multiple simultaneous same-date tabs still share a draft key; this change does not establish independent backup histories. Autosave/page-hide network saves, conflict-resolution choices, timezone handling, and the remaining Phase 2/3 acceptance work are still pending. Google remains disabled.

## Unverified assumptions

- S2 Google Drive folder metadata behavior is verified for the test account; production adapter behavior remains unverified.
- Theme and browser-specific behaviors remain to be validated in the UI phase.
- Real Google Drive provider reliability and refresh-token handling remain pending human-assisted verification.

## API contract note

- Recommended future practice: define the HTTP API contract with OpenAPI Schema (OpenAPI 3.2.1) once the app reaches the API-routing phase. This keeps route definitions, payload shapes, and status codes explicit and reviewable without changing the current phased implementation plan in the guide.
- The generated OpenAPI server and contract are already in place under `docs/api/openapi.yml` and `internal/web/api/`.

## Verification status

- `go test ./...` passed in the current workspace after the latest UI and preview updates.

## Restructuring verification

- Generated OpenAPI code now lives at `internal/web/api/openapi.gen.go`; the contract source is `docs/api/openapi.yml`.
- `journal.Store` and `journal.Clock` are separate ports in `internal/journal/store.go`.
- Web entry handlers use an injected `journal.Service`; the service owns entry listing, calendar construction, opening, saving, and deletion.
- The local provider passes the shared `internal/journal/storetest` contract suite, including stale-save conflict behavior after deletion.
- `google/wire` was regenerated with `go generate ./cmd/app` after the clock dependency moved to the composition root.
- The expanded depguard configuration passes `golangci-lint run` with 0 issues. Deliberate violation canaries remain a follow-up if the lint configuration needs stronger automated proof.
- Middleware is intentionally only scaffolded at `internal/web/middleware`; implementation is deferred until the relevant feature phases.
- The in-memory cache is created once in `cmd/app` and shared by account-keyed `journal.Store` decorators. It uses a two-minute TTL and a 2048-entry LRU bound.
- Cache tests cover TTL expiry, capacity eviction, account isolation/shared reuse, preview-size keying, and mutation invalidation.
- `journal.Service.Page` reads newest year indexes only as far as needed to fill a page and detect `HasNext`, then fetches previews for that page only.
- `journal.Service.Month` uses one year index for entry markers rather than issuing one `Get` per calendar day.

## Localization

- `assets/locales/en.json` is the canonical source for user-facing UI copy.
- The locale source uses flat, dotted string keys such as `landing.feature.quiet.title`; `internal/i18n` loads it directly as a string map.
- The same flat catalog is embedded for browser scripts, so browser and Go template lookups use identical keys without load-time flattening.
- `internal/i18n` loads the embedded catalog once and provides message lookup plus the locale's date layout.
- Go templates call the `t` template function. The base layout embeds the JSON catalog in an escaped `data-messages` attribute, which `app.js` and `theme.js` read for status text, calendar labels, list states, and theme-toggle accessibility text.
- API handler errors and generated OpenAPI parameter errors use catalog messages instead of exposing raw Go/provider error text.
- Missing keys fall back to the key itself to make incomplete translations visible during development.
- Account email, journal dates, and user-written entry previews are runtime data rather than translatable interface copy; their surrounding labels are catalog messages.
- New user-visible template, JavaScript, or handler copy should add a key to `assets/locales/en.json` and use the key at the point where it is rendered.

## Authentication boundary and startup validation

- Entry APIs require a decryptable session cookie with provider/account identity and an enabled provider; missing or invalid sessions receive HTTP 401.
- Editor, entries, calendar, and settings pages redirect to the landing page without a valid session.
- The request service factory now resolves the store from the session provider and connection, and scopes the cache key by provider/account.
- Configuration parsing rejects invalid ports and `COOKIE_SECURE` values. Validation requires an absolute HTTP(S) base URL, a base64 session key decoding to 32 bytes, a supported log level, and known provider names.
- The local provider can only be registered for exact `localhost` or loopback IP hostnames; `localhost.evil.com` is rejected.
- Startup refuses the Google provider while OAuth and Drive storage remain stubs, instead of allowing placeholder credentials to appear successful.
- Session expiration and key rotation remain open; `IssuedAt` is required but no maximum session age is enforced yet.


## Browser date consistency (2026-10-08)

Implemented ADR 010: browser IANA preference resolves once per request, with UTC fallback. Pages, calendar defaults/markers, and future-date checks now share that location. Entry reads and deletes reject future dates before storage access, as saves already do; the API contract documents the actual 400 text response. Regenerated OpenAPI with installed oapi-codegen v2.8.0; generated Go output was unchanged.

Default editor URLs become explicit selected dates before editor requests start. Time-zone correction reloads an explicit URL without changing its date. Saving after midnight retains that date. Browser bootstrap tolerates blocked cookies without repeated reloads; app startup embeds the IANA database.

Verification: `make generate-openapi` and `make fmt lint test` passed (0 lint issues, editor and timezone JavaScript regressions, all Go tests). Fixed-clock cases include Jakarta midnight, Los Angeles, Kiritimati/Honolulu, and New York spring/fall DST transitions. Router tests verify local-day access and UTC fallback; middleware tests verify invalid input and one resolution per request. These are automated observations, not live browser acceptance.

Human-reported check (2026-10-08): `/entries/today` immediately became `/entries/2026-10-08`. This confirms default-route date canonicalization in the tested browser; browser/version was not specified for this check. The user subsequently confirmed that reloading an explicit past entry preserves its date and the calendar marks October 8 as today. Timezone cookie/correction was subsequently confirmed on 2026-10-09; broader browser coverage remains limited to the reported checks.

Manual follow-up: run local mode with `make run`; inspect `mj_tz` in Chrome and Safari, verify correction when the browser timezone changes, and check current-day navigation. Default-route canonicalization, past-date reload, and the today calendar marker have passed in the user-tested browser. Selected-date preservation was accepted by the user based on the observed date-specific PUT request; automated tests cover the midnight crossing. A loaded page retains its initial today snapshot until navigation/reload. Configurable week start remains pending.


Human-reported midnight check (2026-10-09): at 00:00 WIB (Asia/Jakarta), the entry date became `2026-10-09` and the calendar marked October 9 as the current day. Browser/version and navigation method were not specified. This confirms the observed current-day selection and calendar marker at local midnight; the user subsequently accepted selected-date preservation based on the observed `PUT /api/v1/entries/2026-10-09` request. The request targets the selected record date. An already-open prior-day save across midnight is covered by automated domain tests; the reported API observation does not independently demonstrate that particular live scenario.


Human-reported browser/layout checks (2026-10-09): Safari testing and simulated mobile screen-size changes looked good. Exact scenarios and browser version were not specified in this report. This records Safari and responsive-layout observations; actual phone-browser testing was subsequently skipped by user decision (see below). Time-zone correction was subsequently confirmed as recorded below.


Human-reported time-zone correction (2026-10-09): after the stale-cookie test, the user confirmed that the timezone cookie corrects itself and `/entries/today` selects the correct date for `Asia/Jakarta`. Browser/version was not specified for this check. This completes the reported manual cookie-correction and local-today checks; actual phone-browser testing was subsequently skipped by user decision; configurable calendar week start remains pending.


Acceptance decision (2026-10-09): the user chose to skip actual phone-browser testing and accept browser developer-mode device-toolbar testing for this stage. This records a waived device check, not evidence of a physical phone run.


## Week-start deferral and pagination bounds (2026-10-09)

The user explicitly shelved configurable calendar week start as a nice-to-have feature. This supersedes the guide's current-phase requirement for both week starts. Keep Monday first; the idea is parked in `docs/development/ideas.md` and is no longer an acceptance blocker.

Pagination previously allocated all dates up to the requested offset and multiplied unchecked page values. The service now accepts pages 1–10000 and sizes 1–100, validating before storage access. It skips offset entries and retains at most size+1 dates to detect the next page. API defaults remain page 1/size 20; explicitly invalid ranges return localized HTTP 400 instead of silently defaulting. The API contract records those bounds and responses. Year metadata still comes from the provider's year index; this change bounds retained page dates, not provider index allocations or total scan time.

Verification: `make generate-openapi`, `make fmt lint test`, and `git diff --check` passed. Lint reported 0 issues; editor/timezone JavaScript regressions and all Go tests passed. New tests cover maximum-int and invalid ranges, cross-year ordering, an empty high page without preview reads, and API 400 mapping. Generated Go output was unchanged. Full API error-contract alignment, atomic local writes, and session/security lifecycle work remain separate acceptance tasks.


## User editor-recovery acceptance report (2026-10-09)

User reported checks 1–6 passed: normal/empty draft restoration, save snapshot state while typing, offline retry, retained conflict text, and retained text after HTTP 401. Reconnecting to local in another tab allowed the original tab to save. Conflict reload deliberately retains its stale base version; conflict-resolution choices remain open. Browser/version was not specified for this report.

Check 7 was inconclusive: blocking all site data prevented cookie-based login and produced 401. That does not isolate draft-storage failure. The checklist now provides a console override for draft writes that preserves cookies. Check 8 passed theme retention and the reported remaining sign-out checks; Appearance controls were observed to be inert and stuck on System and remain an open UI issue. Check 9 was not yet performed; explicit synthetic legacy-key steps are now documented.

Check 10 partially passed: Save was disabled during GET, but the user observed typed text replaced on response. Existing mocked input-event tests preserve such text, so the actual cause is not established. Following the user's suggestion, the initial HTML textarea is now read-only until a valid GET succeeds; failed GET keeps it read-only and offers Retry. Typing during PUT remains enabled. Defensive input preservation remains tested. The initial-load lock was subsequently confirmed in the user retest recorded below.

`make fmt lint test` passed (0 lint issues; all JavaScript regressions and Go tests). A reconnect action on 401 is a reasonable future auth flow, provided drafts are preserved and account identity is rechecked; no automatic Google redirect was added while production Google remains disabled.


## Recovery checks 7 and 9 passed (2026-10-09)

Human-reported check 9: the synthetic legacy backup produced Review older draft; cancel preserved server text; confirm restored the synthetic text as unsaved; reload recovered it with Draft restored. This verifies the reported explicit legacy recovery and subsequent scoped-backup restoration. Legacy-key removal was not separately reported.

Human-reported check 7 with draft writes blocked and cookies enabled: editing displayed Unsaved changes plus the unavailable-backup warning; online Save succeeded. An offline save displayed Save failed plus the same warning, without claiming a local backup. Reload lost the unsaved edit, as expected when the backup write failed. The warning accurately instructs the user to keep the page open or copy the text. Browser/version was unspecified. Both checks are accepted; check 10 was subsequently confirmed as recorded below.


Human-reported check 10 retest (2026-10-09): the initial-load lock works normally. After loading finishes, typing remains enabled during a slow save; after acknowledgement, newer text correctly shows Unsaved changes. This accepts the reported initial-load lock and in-flight save interaction. Browser/version was unspecified. The failure/retry branch is covered by automated tests but was not separately reported in this retest. The focused recovery checks are accepted for the reported scenarios; Appearance controls, full conflict actions, and same-date tab draft isolation remain separate work.


## Conflict actions, tab drafts, and Settings navigation (2026-10-09)

Implemented the guide's three conflict choices. Keep mine fetches the latest version and retries the ordinary conditional PUT; Load the other version confirms before replacing local text; Keep both appends latest saved text below local text and requires review/Save. Restored stale-version drafts expose the actions immediately. Failed resolution retains text and allows retry, and further concurrent changes can conflict again. Existing provider atomicity limitations are unchanged.

The user chose reload recovery without requiring fresh-tab recovery after closure. New `mj-draft:v3` backups use sessionStorage so same-date tabs no longer overwrite or clear each other's drafts. Old account-scoped/local date-only backups require explicit recovery and retain their base versions where known. Successful saves clear only that tab's backup. Sign-out broadcasts cleanup to other open editors. ADR 011 records session lifetime and cleanup limits.

Removed inert Appearance controls from Settings as requested; the shared theme toggle remains available. Signed-in pages expose Settings in the shared header. New copy uses the locale catalog. Physical-phone checks remain waived; device-toolbar layout checks for this added link are still needed.

Added interaction regressions for all conflict choices, cancellation, failed resolution, a second conflict, deleted-entry restoration, typing during Keep mine save, tab isolation/reload, and old scoped-version migration. The JSC test runner now requires a suite success marker: inspection revealed that its asynchronous assertion failure could otherwise leave a successful shell status. New manual acceptance steps are in the recovery checklist. The user subsequently confirmed all eight manual checks passed, as recorded below.

Verification for this change: `make fmt lint test` passed with 0 lint issues, both JavaScript suites, and all Go tests; `git diff --check` passed. Tests use distinct per-tab stores with shared local storage and check that saving one tab leaves the other backup intact. Settings rendering verifies shared navigation/theme controls and removal of Appearance. Header actions can wrap at narrow widths. The reported conflict actions, tab-backup reload behavior, sign-out cleanup, and navigation checks subsequently passed, as recorded below.

Settings header link refinement (2026-10-09): replaced the visible Settings label with an inline outline gear. It shares the theme toggle’s circular styling, hover/focus treatment, and theme color variables. The SVG inherits currentColor, and the link retains a localized Settings accessible label and tooltip.

Gear geometry refinement (2026-10-09): replaced the uneven outline with six identical radial tooth profiles repeated at 60-degree intervals around (12,12). Inner/outer radii and center ring are concentric; existing currentColor and header styling are retained.


SVG asset extraction (2026-10-09): the HTML scan found three inline icons, all in base.html (sun, moon, gear). Moved their unchanged geometry to assets/static/svgs. Templates now use decorative spans, and CSS masks render the standalone SVGs using currentColor, preserving light/dark colors and theme icon visibility. Existing embedding includes the directory automatically. Button/link accessible labels remain on their controls.


## Conflict/tab-backup browser acceptance (2026-10-09)

The user reported all eight checks in the Conflict actions and independent tab backups checklist passed:

- Same-date tabs retain independent drafts on reload; saving one does not clear the other backup.
- Saving A then B produces a conflict in B; reloading B retains its text and the conflict controls.
- Keep mine in B succeeds; reloading clean A shows B's saved text.
- Load the other version cancel preserves text; confirm loads the saved version with GET only, without PUT.
- Keep both combines the texts and shows Unsaved changes for review.
- The delayed/failed-resolution scenarios in check 6 passed as reported, without further per-scenario details.
- Sign-out cleanup and stopping draft recreation in other editors passed.
- Settings navigation, shared theme control, and the checklist's layout checks passed.

These are human-reported results; browser/version was not specified in this report. They accept the focused conflict/tab-isolation/Settings checks, not the remaining security/API/provider acceptance backlog. Actual physical-phone testing remains waived by user decision.


## PR #3 review: failed logout backup loss (2026-10-09)

Reviewed the submitted review and discussion thread `discussion_r4226317173` against head `9757a2c`. One unresolved P2 finding was present and is valid: cleanup/broadcast occurred before POST logout, so a rejected/failed request could erase backups and mark other editors signed out despite no confirmed logout. The review summary reported an independent 503 reproduction; this session verified the ordering in the code and added targeted regressions.

Fixed locally: perform a non-draft storage-access probe, request logout, then broadcast final sign-out and clear backups only after a successful response. Pending, HTTP 503, and network-failed logout retain tab/legacy backups and do not publish the sign-out marker. Editors can recover and save afterward when the server session remains active. Confirmed logout followed by storage-cleanup failure gets a distinct signed-out warning and a cleanup-only retry. A failed transport may follow server-side cookie clearing, so retaining drafts does not assert that authentication always survived.

Regression coverage includes pending logout, 503/network failures, restored-editor saving, successful retry, and cleanup retry without another POST. Browser retest is documented separately. No review replies, thread resolution, commits, or pushes were performed; the review finding remains open until the fix is published and assessed.

Verification of the review fix: `make fmt lint test` passed with 0 lint issues, both JavaScript suites, and all Go tests. `git diff --check` passed. These are local automated checks; the user subsequently confirmed the failed-logout browser checklist works, as recorded below.


Human-reported failed-logout acceptance (2026-10-09): the user tested the Failed logout recovery (PR #3 follow-up) checklist and confirmed it works. Browser/version and optional HTTP 503 interception details were not specified; those branches remain supported by automated regressions. The user will commit the changes to the PR. No commit, push, or remote review-thread resolution was performed by the assistant.
