# Findings Log

## Authentication-expiry recovery (2026-10-13)

Previously, save/delete distinguished 401 but offered no re-login action; Entries/Calendar displayed generic failures. A production request log establishes only the status, not whether the cause was cookie validation, deployment-key changes or Google access. That cause remains unverified.

Regression tests cover 401 during initial entry load, save, delete, conflict resolution and disconnect; retained text/backup, continued typing, paused idle/blur/pagehide writes, blocked storage feedback, and fresh verification before manual save retry. List and calendar tests cover shared feedback, successful retry, and obsolete calendar failures. Go tests cover login availability with or without a valid cookie, no session-cookie clearing, no-store/noindex headers, language return paths and mismatched account scopes rejected before service construction. The optional account-consistency header is documented in OpenAPI; regeneration produced no generated API changes.

Fresh `make fmt lint test` passed with zero lint issues, and `go test -race -short ./...` passed. All 36 sequential architecture probes and 12 Python helper regressions passed. Local headless Chrome with axe passed 50 scans, 44 keyboard checks and zero page errors in light/dark themes, including expired-authentication states, a 390px viewport without horizontal overflow, and signed-in/signed-out `/sign-in`. A real local-provider new-tab reconnect preserved the original draft and an explicit retry saved it; synthetic 401 responses exercised expiry. Screenshots were inspected locally. Report: `/tmp/myjournal-auth-expiry-a11y.json`. Axe retains incomplete aria/contrast checks; these results do not establish native screen-reader announcements, real Google reconnect behavior or production deployment health. The QA process owns an ephemeral listener and temporary synthetic data and cleans both up.


## Test tooling validation (2026-10-13)

Testify v1.12.1 and Mockery v3.8.1 were verified with the declared Go 1.27.2 toolchain. Twelve generated `mock_*_test.go` files reproduce byte-for-byte with `make generate-mocks`. Mockery remains a pinned development command, outside the application's dependency graph; Testify is imported only by tests. Typed expectations verify relevant call arguments and reject unexpected calls, including exchange after invalid login callbacks and revocation for rejected requests. Explicit stateful fixtures remain useful for storage semantics, cancellation and concurrency; this is not a mechanical replacement of every test assertion or fixture.

Fresh local validation passed: `make fmt lint test` (zero lint issues), `go test -race -short ./...`, `go vet ./...`, actionlint v1.7.7, nine PR-selector Python regressions, generation reproducibility and `git diff --check`. The sequential architecture suite passed all 36 probes, including new production-Testify and core-test/driven-adapter violations, and removed its temporary files. Installed-binary Mockery regeneration, `wire diff ./cmd/app`, actionlint and all 12 Python helper regressions also passed before PR preparation. Hosted CI has not run these changes. No live Google or browser-manual acceptance was repeated for this test-tooling change. The pre-existing Makefile coverage command is preserved; `make test` regenerates its whole-repository coverage profile.


## Production function coverage (2026-10-13)

The user selected a target of at least 85% statement coverage per maintained production function. Generated API/Wire construction bodies, test-support packages and historical OAuth spikes are excluded from that function target. Handwritten composition helpers emitted into wire_gen.go remain included. The profile itself still includes generated production code; no counters were suppressed or manually rewritten.

The initial supplied coverage.out reported 61.3% across the wider repository, including historical experiments, and used package-local instrumentation: functions tested from other packages could appear as 0%. A fresh comparable production baseline with cross-package instrumentation reported 84.7%. The same production measurement after the additions reports 90.4%.

| Comparable production measurement | Before | After |
| --- | ---: | ---: |
| Statement coverage (profile total, including generated production code) | 84.7% | 90.4% |
| Maintained functions below 85% | 49 | 5 |
| Maintained functions at least 85% | 191/240 | 236/241 |

The extra function is the private atomic staging helper. Its file port models actual Write/Sync/Close side effects, allowing tests to inject disk-stage failures and verify that old text survives, replacement never commits and temporary files are removed. It preserves the original os.CreateTemp/write/sync/close/rename ordering and public Store behavior.

Other regressions exercise explicit startup environment parsing, occupied-listener and shutdown errors, session-scoped service construction, registry lookup/error privacy, invalid journal paths/dates, lock cancellation before and during handoff, partial prefix-scan cancellation/retry, malformed filesystem indexes, canceled/failed trash preservation, codec entropy/shape failures, Drive error translation and changing metadata/checksum responses, timing-body read/close errors, calendar/delete/settings HTTP mapping and the health envelope. Filesystem permission tests explicitly skip when root would bypass permissions; cancellation tests also cover trash preservation independently of permissions.

### Remaining per-function shortfalls

| Function | Coverage | Remaining limit |
| --- | ---: | --- |
| cmd/app/main.go: main | 0.0% | Process entry point calls os.Exit; covered startup/error/shutdown logic lives in run/serve. Exercising main itself belongs to a subprocess integration-coverage run. No exit hook was added solely to increase unit coverage. |
| cache/local/bigcache: New | 83.3% | Remaining branch handles a library constructor error after fixed valid shard/config values and positive TTL validation. No mutable factory hook was added to force an otherwise unreachable library result. |
| cache/local/bigcache: DeletePrefix | 81.2% | Cancellation, retry and account isolation are tested; remaining branches handle iterator retrieval/deletion errors during library cleanup races or unexpected underlying failures. No BigCache-internal corruption or timing-dependent test was introduced. |
| web/discovery: registerDiscoveryRoutes | 84.6% | The uncaught coverage branch is xml.Marshal failure for the fixed sitemap structure containing strings. Valid production input cannot trigger it. |
| web/i18n: Catalog.JSON | 75.0% | The defensive json.Marshal error branch cannot be triggered by its concrete map[string]string payload. |

These five are visible in the profile; the requested per-function threshold is not fully achieved. Protective error handling was retained rather than removed to change percentages. Coverage is statement coverage, not proof of all branches, race outcomes or production correctness.

### Reproduction and verification

```sh
go test -coverpkg=./cmd/app,./internal/... -coverprofile=coverage.out ./cmd/app ./internal/...
go tool cover -func=coverage.out
```

coverage.out was regenerated with that command. The original supplied profile is preserved locally at /private/tmp/myjournal-coverage-before.out. The user's Makefile default still runs package-local coverage over ./...; running make test overwrites coverage.out with that broader profile, so regenerate the production profile above for the comparable report.

Fresh checks passed: make fmt lint test (zero lint issues), go test -race -short ./..., the production coverage run and all 34 sequential architecture probes. The final test-only logger-restoration cleanup was verified by the full formatting/lint/test run. Architecture probes ran after builds/tests and removed their temporary files. Final git diff --check passed. Logs: /private/tmp/myjournal-coverage-verified.log, /private/tmp/myjournal-coverage-race.log, /private/tmp/myjournal-coverage-maintained.log and /private/tmp/myjournal-coverage-architecture.log. No live Google calls, new deployment, commit or push were performed.


## Architecture refactor verification (2026-10-13)

Implementation status and remaining scope are recorded in [Progress](PROGRESS.md#architecture-audit-implementation-2026-10-13); [ADR 021](adr/021-delivery-ports-and-login-capabilities.md) records port ownership and capability policy.

- Production import inspection with `go list` confirms that no package under internal/driver imports a driven package. Codec and connection contracts are satisfied structurally and composed by Wire.
- Synthetic historical ciphertext fixtures decrypt through the injected Codec. Regression tests retain session/login purpose separation, wrong-key rejection, fixed lifetimes and callback validation. Focused fakes exercise codec failure, connection failure, cookie preservation and use of Catalog without storage construction. Login-flow tests use provider names independently of the declared capability to prove names no longer decide validation requirements.
- An AST comparison against the pre-refactor Google store.go confirms all 25 original function declarations are unchanged across store.go, folders.go, read.go, write.go and retry.go. This establishes the file split's function-level equivalence; it does not replace adapter contracts or prove external Drive atomicity.
- Fresh validation passed: `make generate-wire`, `make fmt lint test` (zero lint issues), `go vet ./...`, `go test -race -short ./...`, focused non-short app/core/session/provider/cache/web race tests, all 34 sequential architecture probes, 12 Python regressions and `git diff --check`. Architecture probes were kept separate from lint/build/test work and cleaned up their temporary files. Main outputs are `/private/tmp/myjournal-hex-final-checks.log`, `/private/tmp/myjournal-hex-final-race.log` and `/private/tmp/myjournal-hex-architecture.log`.
- All 18 local .omm diagrams validate without errors; structural checks verify 95 node descriptions, existing source paths and labeled edge endpoints. The existing node-count heuristic warnings remain on the route/editor views. No diagrams were uploaded.
- No new live Google, deployed-version, visual accessibility or performance acceptance was run. Existing session/cookie compatibility, shared cache policy and process-local concurrency limitations remain in force. No commit, push or deployment was performed for this refactor.


## Clean and hexagonal architecture audit (2026-10-13)

This audit describes the pre-refactor snapshot. Its recommendations and lower-priority refinements are now implemented; [current status](PROGRESS.md#architecture-audit-implementation-2026-10-13) and [ADR 021](adr/021-delivery-ports-and-login-capabilities.md) supersede the codec exception, concrete connection dependency and provider-name branch described below.

Verdict: the current implementation substantially follows both clean architecture and ports-and-adapters principles. This is supported by the production import graph and use-case/adapter implementations, rather than folder names alone. No forbidden core dependency or infrastructure call inside journal/connection use cases was found in the inspected code. The main qualification is an intentional delivery-to-session-codec dependency allowed by ADR 018 and depguard; this is a replaceability gap under a stricter ports-only interpretation, not a newly discovered violation of the project's current rules.

Scope: the seven local `.omm/` perspectives and recursive diagrams; `cmd/app`, all production `internal` package imports, core services/ports, web/session delivery, Google/local providers, journal cache/storage, relevant contracts/tests and architecture enforcement. This is an architecture audit, not a new security assessment, live Google acceptance, performance benchmark or exhaustive correctness proof. Local `.omm/` artifacts were not changed or uploaded during this audit.

### Conformance evidence

| Concern | Current implementation | Assessment |
| --- | --- | --- |
| Domain/application independence | `internal/core/journal/service.go` injects Store/Clock, validates dates/text/pagination and uses plain domain results. `connection.Service` consumes Catalog/Gateway, Clock and entropy. | Established; no HTTP, filesystem, OAuth SDK or concrete adapter imports in core. |
| Outbound ports | `journal/store.go` owns Store/Clock and optional scoped/fresh capabilities; `connection/ports.go` owns lifecycle Gateway/Catalog. | Established; infrastructure implements the consumer's contract. |
| Transport mapping | `handler/entries.go` maps generated dates/JSON to domain types and domain errors to HTTP; router and auth handlers own redirects/cookies. | Established; HTTP details stay out of use cases. |
| Runtime composition | `cmd/app/wire.go` constructs providers/cache/clock and injects session-scoped JournalService through a factory; generated Wire owns emitted construction code. | Established for providers/cache/clock; codec injection is the exception below. |
| Storage substitution | Google and local providers implement Store; cache decorates Store; BigCache/LRU implement ByteCache without changing domain policy. | Established; adapter memory/concurrency semantics remain different. |
| Independent policy tests | Core uses fake ports. Google fake-HTTP and local temporary-filesystem tests run shared Store contracts; cache wrappers also run them. | Established within automated/synthetic scope. |
| Enforced dependencies | `.golangci.yml` rejects outward imports recursively; temporary probes include nested packages and test-support leakage. | Fresh lint and all 31 probes passed. |

`cloud.google.com/go/civil` in core is a calendar-value dependency, not a Drive client or network dependency. Context-aware ports, operation-scoped discovery and explicit fresh reads do not make the domain HTTP-dependent. Combining entities/use cases in one small feature package is also compatible with this design; separate domain/application folders are not required here.

### Recommended improvement 1: inject the session codec at the delivery boundary

Priority: worthwhile boundary improvement; not an urgent correctness fix.

Evidence: `internal/driver/web/authsession/session.go:10` imports driven/session and Read calls Decode at line 26. `internal/driver/web/handler/auth.go` imports that same concrete package for EncodeLoginFlow (line 40), Encode (line 89) and DecodeLoginFlow (line 136). `.golangci.yml:78` explicitly permits this cross-adapter edge. ADR 018 records encryption as an HTTP concern while the algorithm lives in the driven adapter.

Impact: changing the session encoding implementation requires changing web code; HTTP tests cannot replace encoding independently. The boundary also passes the raw session key through delivery configuration because encryption is called as package functions.

Recommendation: define a small session/login codec contract owned by the consuming web/session delivery package, implemented by driven/session and constructed once in cmd/app. Keep cookie names, paths, attributes, HTTP response mapping and encoded-size limits in web. Keep session models/lifetimes in connection core and cryptography in driven/session. Update depguard to remove the exception only after the injection seam is complete. Do not invent a server-side session repository/database or move HTTP cookie policy into core.

Validation for a later implementation: retain ciphertext compatibility fixtures, purpose separation, expiry/provider validation, malformed-cookie and login callback regressions, plus integration tests using the real AES-GCM implementation. Add focused handler tests with fake codec failures. No implementation is authorized by this audit alone.

### Recommended improvement 2: give connection delivery an inbound use-case seam

Priority: useful testability improvement; existing concrete core dependencies are legal.

Evidence: `internal/driver/web/handler/server.go:41` stores `*connection.Service`, and its constructor takes the concrete service at line 50. Journal operations already consume the local JournalService interface at lines 24–34. `router.go` likewise takes the concrete connection service. Handler composition tests construct the real connection.Service and gateway registry to exercise lifecycle behavior.

Impact: isolated HTTP tests for settings, revocation failure and login completion must arrange core gateway behavior instead of directly supplying the use-case outcome. This does not prevent testing or violate dependency direction, but it makes the inbound boundary less replaceable than the journal boundary.

Recommendation: introduce one consumer-owned connection use-case interface containing the methods delivery actually calls: Begin, Complete, Disconnect and Settings. Keep connection.NewService returning its concrete type and bind it in cmd/app. Where delivery only checks whether a provider is configured, consider the existing narrow Catalog.Lookup capability rather than exposing the storage-capable Registry.Get surface. Keep real-service integration tests; do not add interfaces around every renderer/config/value type.

### Lower-priority observations

- **Provider-specific login policy is explicit but not fully generic.** `connection/service.go:59` and `:86` special-case LocalID; other providers are expected to use the pending browser login transaction. This is reasonable for the locked Google/local scope and keeps state/PKCE/nonce validation in application policy. Before introducing a genuinely different login protocol, model that capability explicitly rather than extending provider-name conditionals. Additional providers remain deferred; no capability framework is needed now.
- **Google Store has several cohesive sub-responsibilities in one file.** `internal/driven/provider/google/store.go` is 557 lines and contains client construction, folder discovery, reads, mutations, preview work, retries and error mapping (notably retryRead at line 481 and mapError at line 517). If concurrent edits or maintenance justify it, split into focused files inside the same package. Do not create another service/repository hierarchy or move Drive mechanics into core merely to reduce file size.
- **The dependency diagram understates an existing edge.** `.omm/dependency-map/diagram.mmd` shows driving-to-core and driven-to-core but omits web-to-driven/session as an arrow; its node description mentions the exception. A future diagram update should draw that codec dependency explicitly, and distinguish byte-storage-to-cache-policy ByteCache ownership from provider-to-core dependencies. Diagrams are a guide; go list imports remain the audit evidence.

### Changes not recommended

Keep the current core/driver/driven structure, concrete core constructor returns, handler-owned JournalService, shared journal cache decorator and consumer-owned ByteCache. Do not move caching into Google, relocate domain rules to handlers, create an interface for every struct, replace civil.Date solely because of its module name, or split Clock into a new generic utilities package. Preserve two-minute snapshots, fresh recovery reads, generation guards, bounded singleflight/warming and same-process coordination limits.

### Fresh verification and limits

- `go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./internal/... ./cmd/app`: inspected production dependency graph; the only direct driving-to-driven edge is session encryption in authsession/handler.
- `make lint`: passed, zero issues. Output: `/private/tmp/myjournal-architecture-audit-lint.log`.
- `make test-architecture`: all 31 probes passed, run after lint and before package tests; temporary files were cleaned up. Output: `/private/tmp/myjournal-architecture-audit-probes.log`.
- `go test ./internal/core/... ./internal/driven/cache/... ./internal/driven/provider/... ./internal/driven/session/... ./internal/driver/web/...`: passed; Go reused cached test results where applicable. Output: `/private/tmp/myjournal-architecture-audit-tests.log`.
- No production code, interfaces, generated files, runtime configuration or diagrams were changed by this audit. No commit, remote write or deployment was performed. An independent user change to `.gitignore` was preserved.


## Current status (2026-10-12)

This summary supersedes pending language in the dated evidence below; those records describe what was known at the time, not new work to repeat.

- Google OAuth/Drive registration and storage are implemented. Non-test registration is allowed with a valid HTTPS origin and Secure cookies; GOOGLE_TEST_MODE remains restricted to exact loopback/localhost URLs. Restricted Google Testing acceptance and the first v0.1.0 published-release deployment passed on 2026-10-10 (ADR 016). This is not unrestricted Console publication or branding verification.
- Theme initialization is already externalized to assets/static/js/theme-init.js, loaded before CSS. Same-origin CSP/security headers and host-only HTTPS HSTS are implemented (ADR 012). There is no remaining inline-theme migration task.
- Focused Chrome UI acceptance is complete with separately recorded human announcement evidence and device-testing waiver; coverage limits remain in [UI acceptance checks](ui-acceptance-checks.md). Full WCAG certification and blanket Phase 2/3 completion are not asserted.
- Google branding/ownership submission remains separate; see [branding verification](google-branding-verification.md). Accepted latency/revision-retention limits and non-atomic external-writer/deployment-overlap risks remain documented rather than treated as passed targets. No new Google experiments or deployment are implied by this reconciliation.

- English and Bahasa Indonesia are implemented with public language preferences, a globe/code header pill, an always-visible Settings gear and Help; Privacy is reached from Help. Account actions still require a valid session. Earlier English-only and conditional-Settings notes are historical.
- Current footer/release/logging and date-format behavior is consolidated in [the implementation summary](#footer-release-metadata-and-date-formatting-2026-10-12). Final footer visuals and Docker/deployed-version checks remain unverified; no new release is implied.

## DELETE to Entries diagnostic (2026-10-13)

A temporary synthetic Drive diagnostic seeded eight entries across two years, deleted one date, allowed backend warming to finish during a simulated five-second wait, and requested the latest ten entries. Twenty race-detector runs passed. The diagnostic file was removed afterward; its output is retained locally at `/private/tmp/myjournal-delete-list-diagnostic.log`. No runtime behavior was changed.

| Before DELETE | First list metadata requests | First list content downloads | Immediate repeat requests |
| --- | ---: | ---: | ---: |
| Entries list already loaded within TTL | 3 | 0 | 0 |
| Entries list/content cold | 4 | 7 | 0 |

DELETE warming itself used three metadata requests and no downloads to confirm absence. It calls ordinary Get only. DELETE invalidates the shared year list and affected year index, so waiting does not repopulate those index snapshots. The next Entries request still traverses ListYears/ListYear; surviving full-entry snapshots do avoid content downloads when already present. Metadata discovery within warming is operation-local and does not populate the decorator's retained index snapshots.

This demonstrates a coverage gap in the warming policy, rather than proving the cache is globally broken or diagnosing the user's live timing. A potential follow-up is to warm ListYears and the affected ListYear before Get, through the shared decorator and a fresh operation, after membership-changing mutations. That would target invalidated indexes without seeding acknowledgement data. Newly visible or previously uncached entries would still require content reads. The follow-up is now implemented: creation and trash warm both indexes through the ordinary decorator before Get. Regression tests prove post-warm index hits, existing-entry update scope and foreground index singleflight sharing. The synthetic Google fixture retains three metadata requests per warm and subsequent detail/index reads make no upstream requests. Live latency, expiry/eviction and process identity remain relevant to actual hit rates. Follow-up validation: `make fmt lint test`, cache/Google race tests, all 31 architecture probes and `git diff --check` passed.

## LRU adapter compatibility (2026-10-13)

- The mismatch was in construction/lifecycle, not ByteCache methods: LRU previously returned one value from New, accepted nonpositive TTL and lacked Close. Regression tests verify invalid TTL rejection, repeated Close, released entries, explicit post-close errors and concurrent Close/operations. Existing ownership, exact expiry, LRU eviction and cancellation tests remain.
- Validation: `make fmt lint test`, focused app/cache/Google race checks and all 31 sequential architecture probes passed; temporary probes were removed. `git diff --check` passed.
- Shared journal Store contracts run through LRU with construction errors checked and storage closed after use. Google synthetic fixtures likewise own/close LRU storage. Both adapters support the same construction/warming/cleanup shape; current wire.go and generated startup select BigCache. Capacity policy remains adapter-specific; no production memory/latency improvement is claimed.

## Mutation-triggered cache warming (2026-10-13)

- Google's OAuth token source captures the Store construction context. A new operation view alone cannot detach it; warming therefore constructs a new provider client with its own bounded context. The synthetic Google test cancels the original PUT request and evicts its access token before releasing warming, forces token refresh through the independent client, and proves the next detail read hits the snapshot; DELETE similarly warms confirmed absence.
- Cache tests gate warming to prove mutation responses do not wait, confirm that stored text/version rather than acknowledgement data seeds content, and verify that ordinary readers join warming while fresh reads stay separate. A later successful mutation coalesces into another attempt and rejects the old fill. Admission tests retain at most 64 dates with four active workers; ten-second attempt expiry and shutdown cancellation/join/queue cleanup are covered. Failed mutations schedule no warming.
- Warming spends optional upstream requests and can be skipped, fail, or finish after navigation begins. The original entry-only implementation did not warm index/calendar snapshots; the list-index follow-up above extends creation/trash warming. These tests do not establish live latency gains, atomic external-writer safety or cross-process coherence. See [request counts](performance-benchmark.md#mutation-triggered-cache-warming-2026-10-13).

## Shared entry/preview content (2026-10-13)

- Complete-file tests cover meaningful text beyond 64 KiB, the 1 MiB boundary/overflow, invalid trailing UTF-8, missing/mismatched checksum and retry after failed validation. Both raw Google and decorated storage pass shared Store contracts.
- Operation-local read discovery preserves deterministic duplicate selection and pagination. Two reads use four metadata requests with paginated retained listings, six with an oversized listing, and eight with exhausted query capacity. Bounds remain 16 queries/1000 files; oversized results can be fetched again. A later mutation still detects an externally changed checksum.
- The identical-batch coordinator remains after removal increased metadata work across competing operation views; [ADR 019](adr/019-overlapping-journal-reads.md#shared-content-follow-up-2026-10-13) records the decision. There is no separate retained preview cache. Positive and negative stale-fill guards, fresh-flight isolation, capacity bypass and mutation invalidation are regression-tested.
- Request counts, downloaded bytes and controlled concurrency/timing belong in the [performance record](performance-benchmark.md#shared-entry-and-preview-content-2026-10-13). Live Drive latency, external-writer atomicity and cross-process invalidation are not established by these tests.

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

## Self-contained contributor guide (2026-10-09)

Rewrote root AGENTS.md to distill product requirements, architecture/generation decisions, UI/accessibility expectations, and phase acceptance without links or dependencies on private planning files. Rechecked the affected source and tracked experiment/ADR documentation. Replaced stale draft/timezone/pagination/conflict/S4/S7 gaps with implemented behavior and reported acceptance; retained actual storage/security/API/autosave/release gaps. Preserved the user decisions to defer week-start configuration, accept device-toolbar coverage, remove Settings Appearance, and use tab-lifetime recovery.

The guide documents failed-logout ordering, tab-scoped backup/migration behavior, external SVG assets, current test tooling, and the next storage-correctness priority. Production Google remains disabled. Also corrected PROGRESS.md's stale browser-timezone blocker to match the recorded passed checks. This is a documentation change; no runtime behavior or private planning files were modified.

Verification: checked the guide’s referenced repository paths, confirmed no private-file references remain, and checked both tracked edits and the untracked AGENTS.md for whitespace errors. Documentation-only; no runtime tests were rerun.

## PR #4 onboarding review follow-up (2026-10-09)

Both nonblocking review comments were valid improvements. README still directed contributors to an unavailable private guide; it now links root AGENTS.md and tracked progress/phase requirements. Its route and Google-status summaries were also aligned with the existing code.

Shortened the always-loaded guide by removing repeated experiment history and detailed phase lists, putting stack/setup/check commands near the top, and retaining critical architecture/product/privacy constraints, user deferrals, text-safety/date/logout invariants, and the current storage priority. The detailed acceptance requirements are preserved in docs/development/phase-acceptance.md; experiment evidence remains in the existing findings/Phase 1 review/ADRs. Repository onboarding still requires no private planning files. No arbitrary line-count validity rule was adopted, and no runtime behavior was changed.

Verification: AGENTS.md reduced from 3,497 words/170 lines to 1,790 words/109 lines. Checked relative links in the guide, README, and phase checklist; all resolve locally and contain no private-file paths. Checked retained critical decisions and whitespace, including the new checklist. Documentation-only; no runtime tests rerun and no commit/push/review-thread resolution performed.

## Local storage mutation safety (2026-10-09)

Rechecked the merged contributor guide and Phase 2/3 acceptance against the local adapter. Put previously checked versions then truncated the destination file without a shared lock. Provider.Store constructs separate store instances, so instance-local locks would not protect simultaneous requests. Trash reused the date as its destination and could overwrite an earlier deleted entry.

Put and Trash now share one bounded process-wide mutex, keeping the version check and replacement in one critical section across all local stores. This intentionally serializes even unrelated local development writes; it does not provide cross-process or external-writer coordination. Canceled mutations are checked after acquiring the lock and before replacement. Waiting for the mutex itself is not cancelable.

Saves stage a private (0600) temporary file in the destination directory, write/sync/close it, then rename it over the entry. Readers see a complete old or new file. Pre-rename failures preserve the original and clean up the staged file. This is atomic replacement, not a claim of power-loss durability: the directory is not fsynced, and process termination may leave an unreferenced temporary file. Trash reserves a unique date-prefixed destination, preserving prior copies, including legacy date-only trash files.

New regressions cover twelve simultaneous stale-version saves using separate stores (exactly one succeeds), an already-open reader retaining the old complete contents after replacement, injected rename failure preserving the original and removing staging files, repeated trash/recreation preserving all copies, and canceled mutations preserving content. Verification: `make fmt lint test` passed (0 lint issues, both JavaScript suites, all Go tests); `go test -race ./internal/provider/local` passed. No browser acceptance or live Google behavior is claimed. UTF-8 validation/previews and the other Phase 2/3 gates remain open.

## UTF-8 and initial session/request protections (2026-10-09)

Save now rejects invalid UTF-8 before domain/storage mutation. The local adapter also rejects malformed files on reads and conditional updates rather than silently replacing their bytes; previews omit only missing entries and propagate other failures. Raw API JSON is validated before decoding because encoding/json otherwise replaces malformed bytes. Unpaired UTF-16 surrogate escapes are rejected as well; valid pairs, literal replacement characters, and escaped backslashes remain accepted. Multiple JSON documents are rejected.

Store previews now trim leading Unicode whitespace and select the first meaningful line before applying the byte budget. Truncation ends at a code-point boundary; it does not promise complete grapheme clusters. Nonpositive budgets return empty prefixes. List requests fetch four extra preview bytes so the displayed 40-byte prefix can retain a truthful ellipsis even when the next character is multibyte. Added domain, local, API, and shared-contract regressions; the complete contract suite now also runs through the real local adapter wrapped in the cache.

Sessions use one 30-day Lifetime constant and ValidAt check shared by pages and APIs. IssuedAt must be present, not in the future, and strictly younger than the limit. The router's injected clock controls both page/API checks and login issuance. Login sets Expires and Max-Age from original issuance; ordinary reads do not renew the deadline. Rotating the encryption key rejects prior cookies. Existing browser cookies without explicit expiry are still constrained by server age checks. Expired APIs return 401 and private pages redirect through the existing recovery-compatible behavior; no new automatic reconnect/navigation was added.

Unsafe browser requests use net/http.CrossOriginProtection with a localized 403 and no trusted-origin bypass. It allows safe methods and headerless non-browser clients by standard-library policy; it is not OAuth state/PKCE validation. Real Google remains disabled. The local development login callback remains a GET; its development-only shortcut is not a production OAuth implementation. Request logging now emits method, query-free path, status, duration, and request ID, omitting queries, headers, cookies, and bodies.

Save reads at most a 1 MiB JSON envelope using MaxBytesReader and returns localized 413 before storage for larger payloads. This limits serialized bytes, including escapes/baseVersion, rather than setting a character count; the browser's existing failed-save handling keeps the draft. OpenAPI records save validation, 401/403, and 413; generation with installed oapi-codegen v2.8.0 produced no generated-code changes because the type/route shapes did not change. Broader error-contract alignment remains pending.

Automated coverage includes Unicode/LF preservation, leading blanks beyond the preview cap, emoji/CJK/combining-mark boundaries, invalid external files, malformed JSON bytes/surrogates, oversize requests, exact expiry/future/zero issuance, old-key rejection, cookie attributes, page/API rejection, cross-origin mutation/logout rejection before cookie changes, same-origin logout, and log secret omission. No new human browser acceptance or Google result is claimed. Remaining security work includes HTTP timeouts/draining, CSP-compatible scripts/headers, real OAuth state/PKCE and provider validation, and complete API alignment.

Verification after the final changes: `make fmt lint test` passed with 0 lint issues, both JavaScript suites, and all Go tests. `go test -race ./internal/journal ./internal/provider/local ./internal/cachestore ./internal/session ./internal/web/...` passed. `git diff --check` passed. No commit or push was performed.

## PR #5 review: explain permanent save-size rejection (2026-10-09)

Checked the review and sole unresolved inline comment, [discussion_r4227283422](https://github.com/mridha-work/myjournal-app/pull/5#discussion_r4227283422), against the current editor. The P2 finding is valid: the server's new 413 response was mapped to generic Save failed, hiding the permanent request-size limit. Text/backups were retained, but retrying the unchanged request could not succeed.

The shared save path now maps 413 to a dedicated locale message stating the 1 MiB request limit and advising copying the full text somewhere safe before shortening and saving again. This also covers saves initiated by Keep mine. Existing draft persistence, base-version handling, unsaved status, and control re-enabling are retained; the UI does not render arbitrary server error bodies.

Editor regressions cover 413 during pending-save edits, retained latest text/base version/backup, leave warning, reload recovery, successful save after shortening, and truthful warning when browser backup is unavailable. `make fmt lint test` passed with 0 lint issues, both JavaScript suites, and all Go tests. `git diff --check` passed. No new human browser result is claimed; the optional browser check is in editor-recovery-checks.md. The fix remains local: no commit, push, review reply, or thread resolution performed.

## HTTP lifecycle and CSP baseline (2026-10-09)

The prior shutdown goroutine called Shutdown, but ListenAndServe returned as soon as listeners closed, allowing main/deferred cleanup to exit while requests were still draining. Startup now opens its listener explicitly and serves with a signal context. Main waits for Shutdown before cleanup; draining has a ten-second bound and closes remaining connections on failure. Startup/listener/shutdown failures propagate to a nonzero exit after deferred cleanup. Added an actual loopback request held open through cancellation to verify that the response completes before serve returns, plus listener-error coverage. Tests use ephemeral owned ports, not the running development server.

Wire's HTTP server now has 5s header, 15s read, 35s write, and 60s idle limits. The existing 30s handler timeout remains; handlers/adapters must still honor cancellation. Wire output was regenerated through `make generate-wire`, not hand-edited.

Moved the existing pre-CSS theme initializer unchanged in behavior to embedded theme-init.js, removing the sole inline template script. Middleware now sets a same-origin CSP without unsafe-inline/unsafe-eval, blocks framing/objects/base overrides, restricts forms, and applies nosniff/no-referrer. Dynamic pages/API/auth/error responses are no-store; static assets retain ordinary caching. No HTTPS-only directive was imposed on HTTP localhost. ADR 012 records the policy/limits. Integration tests cover headers across landing, private-page redirect, 404, API 401, static bootstrap, and cross-origin 403; they verify the bootstrap remains before CSS.

Browser enforcement/theme/editor/calendar acceptance remains pending; existing JavaScript suites do not simulate CSP enforcement. No live Google behavior or full Phase 2/3 completion is claimed. API contract alignment is the next implementation priority. No commit, push, or PR update was performed.

Verification: `make generate-wire` succeeded; inspected the generated diff (timeouts plus the generator-preserved tzdata import). `make fmt lint test` passed with 0 lint issues, both JavaScript suites, and all Go tests. `go test -race ./cmd/app ./internal/web/...` passed. `git diff --check` passed; template scan found no remaining inline scripts, inline styles, or inline click/load/change handlers.

## Live CSP/browser QA (2026-10-09)

Used the explicitly requested Web App QA & Release Copilot workflow against the user's already-running localhost:3030 app. Isolated installed Chrome 155.0.8059.40 headless automation and native Safari 26.4 WebDriver passed local login, theme reload, external bootstrap/gear rendering, synthetic Unicode save/list preview, calendar navigation, reload recovery, conflict/Keep mine, pending-draft sign-out cleanup, security headers, and 401/404 behavior. Both browsers actually blocked an intentionally injected inline script; CSP was not bypassed. Chrome separately passed simulated storage-access failures with truthful backup warnings and both persisted theme choices.

Final evidence: Chrome flow 21/21, Chrome theme/storage 6/6, native Safari flow 25/25 passing assertions. Chrome's 409/404 and inline-probe console messages were expected; no unexpected app CSP errors/page exceptions were captured. Safari action-time events were clean, but full startup-console collection was unavailable. Initial Safari session timeouts cleared after restarting only the QA-owned driver following the user's remote-automation confirmation. Chrome harness locator/string-evaluation issues were corrected without app changes.

Details, exact browser scope, and limitations are recorded in [csp-browser-qa.md](csp-browser-qa.md). Synthetic entries were created only on missing past dates and moved to trash; existing entries were untouched. Temporary browser profiles/sessions and QA-owned Safari driver were closed; the development server remains running. No application changes, commit, push, or full new unit/lint/race run was performed for this QA task. Safari startup-console/blocked-storage verification and broader Phase 2/3/release acceptance remain separate.

## API contract and input safety (2026-10-09)

Regression tests reproduced a destructive save: `text: null` with a valid base version silently became an empty string and overwrote an existing entry. PUT now requires explicit string values for both text and baseVersion. Empty text remains valid; an empty baseVersion creates only a missing entry. Missing/null/wrong-type fields return 400 without mutation. Existing browser saves already supply both strings. Unknown JSON properties retain their prior permissive behavior.

API handlers, generated parameter-binding failures, cross-origin rejections, and API 404/405 routing now use the localized JSON Error envelope. Wrapped domain errors use errors.Is: reconnect requires 401, unavailable storage returns 503, stale writes 409, and unexpected failures 500 without adapter details. Missing entry GET remains 200 with exists=false and empty text/version; delete of a missing entry remains 404. Calendar requires a real YYYY-MM month with a positive four-digit year rather than silently substituting today.

OpenAPI preserves version 3.2.1 and now documents cookie authentication, fixed session lifetime/attributes, actual missing-entry behavior, required non-null save strings, JSON errors, and relevant failure statuses. Removed the unused 422 response; validation remains 400. Public health/sign-in/callback/logout operations explicitly override cookie authentication. Regenerated API types with installed oapi-codegen v2.8.0. This does not enable production Google OAuth or complete remaining Phase 2/3 UI/security acceptance.

Verification: reproduced failing input/error/calendar regressions before implementation, including the null-text overwrite; targeted handler/router tests then passed. `make fmt lint test` passed (0 lint issues, both JavaScript suites, all Go tests), `go test -race ./internal/web/...` passed, and `git diff --check` passed. Router coverage includes generated binding errors, missing sessions, cross-origin rejection, unknown paths/methods, unavailable providers, and wrapped factory failures across entry/list/calendar routes. No new live browser or Google result is claimed.

## PR #7 review: retain allowed methods (2026-10-09)

Verified the sole inline finding against pinned chi v5.3.2: a custom MethodNotAllowed handler bypasses chi's default Allow header generation. Reproduced missing Allow on entry GET/PUT/DELETE, list GET, logout POST, and settings-page GET routes. The handler now probes the registered router with separate fresh route contexts and sets Allow before emitting either the API JSON envelope or page error; no duplicated path-to-method table or handler execution is needed. Regression assertions check exact supported method sets and retain JSON errors for API routes.

Verification: the new allowed-method regression failed on all four routes before the fix and passed afterward. `make fmt lint test` passed with 0 lint issues, both JavaScript suites, and all Go tests; `go test -race ./internal/web/...` and `git diff --check` passed. Changes remain local and uncommitted; no review reply or thread resolution was sent.

## Safe local autosave (2026-10-09)

Implemented the approved five-second idle debounce through the existing conditional manual-save path. Each input resets the timer, unchanged/returned-to-baseline text produces no write, and intentional empty text remains valid. Only one save runs per editor. Newer typing stays backed up and unsaved until a follow-up request, scheduled five seconds after acknowledgement and using the returned version, succeeds. Manual Save cancels queued autosave. No API/provider/dependency changes were needed.

Any HTTP/network/ambiguous-response failure pauses automatic writes, including blur/hidden/page-hide; later typing does not create an automatic retry loop. Manual Save remains available and resumes autosave after success. Explicit successful conflict resolution clears the failure pause but still waits for review: restored/migrated drafts and Keep both output are not automatically saved until editing or Save. Typing while conflicted cannot bypass conflict controls. Found and reproduced a pause-state edge case after an autosave 409 followed by Load other/Keep both; added regressions and clear the pause only after successful resolution. Pending saves and timers respect sign-out and preserve accepted account/date/tab isolation.

Blur flushes eligible dirty edits. Hidden/page-hide first backs up text and attempts fetch keepalive through the same save path; it never duplicates an in-flight save or treats dispatch as acknowledgement. Browser termination, navigation, or keepalive quota exhaustion can still prevent completion. Draft recovery and beforeunload warnings remain until acknowledgement; no guaranteed unload delivery or offline sync is claimed. ADR 013 and autosave-checks.md describe decisions and reproduction steps. Onboarding, Help copy, phase status, and S5 fallback notes were reconciled.

Verification: new debounce/snapshot regressions failed before implementation, then passed. `make fmt lint test` passed with 0 lint issues, both dependency-free JavaScript suites, and all Go tests; `git diff --check` passed. Deterministic tests cover failure statuses 401/409/413/503 and network rejection, failure pause/resume, unavailable storage, recovery review gates, conflict resolution, empty saves, manual cancellation, hidden/page-hide acknowledgement and in-flight preservation, and sign-out cancellation.

Used the Web App QA & Release Copilot workflow for focused automated Chrome 155.0.8059.40 verification. Built using `go build -o /tmp/myjournal-autosave-qa-app ./cmd/app`; ran `/tmp/myjournal-csp-qa-venv/bin/python /tmp/myjournal-autosave-browser-qa.py` against an owned ephemeral-port server with temporary local-provider data and a random session key. Eight grouped checks passed: five-second Unicode autosave/backup clearance, unchanged suppression, held real PUT plus returned-version follow-up, offline failure/reload/manual recovery, conflict/Keep both review, actual focus-blur flush, empty text, and no page exceptions. Browser timers were controlled through Playwright, with real HTTP storage operations. Evidence is in `/tmp/myjournal-autosave-browser-qa.json`; the temporary server/browser were shut down and data removed. No user server or journal was modified. Actual navigation-delivery guarantees, native Safari autosave, and live Google behavior were not verified. S5 synthetic revision measurement remains required before enabling Google autosave. Changes remain local and uncommitted.

## Confirmed trash and list pagination (2026-10-09)

Preserved the user's exact autosavePaused wording. Added Move to trash for existing entries with explicit confirmation that the date's saved entry is trashed and this tab's unsaved text discarded. DELETE locks typing, saving, migration, and conflict controls; cancels queued autosave; and retains drafts until 204. Success clears only this tab/date backup and resets the selected editor to a missing entry, so subsequent typing creates against an empty version. Other tab/date backups remain. Failed/401/404/network/ambiguous outcomes retain editable text, mark even previously clean text unverified, and pause automatic saves; manual Save/retry remains possible. Backup-cleanup failure after successful trash has distinct truthful feedback. No new delete API or provider semantics were introduced.

A regression reproduced blur autosave preempting deletion: clicking Move to trash initiated PUT and disabled the action before click. Explicit Save/Delete focus or pointer activation now takes precedence over blur autosave, including the no-relatedTarget case. Pointer-up/cancel clears that intent. Canceling confirmation leaves the ordinary idle debounce intact; a pending PUT still prevents concurrent deletion.

Entries now uses bounded page queries and full-page Newer/Older links, preserving browser history. Page 1 has no Newer; last/empty later pages retain a path back; page 10000 cannot offer an unsupported page. Failures offer Retry for the same page and Newer while hiding Older. Year grouping/newest ordering remains; previews use textContent directly. Added the dependency-free browsing suite to make test-js alongside editor/timezone regressions, and documented manual checks in delete-browsing-checks.md.

Verification: new deletion/blur/ambiguous-response and pagination regressions failed before their fixes and passed afterward. `make fmt lint test` passed with 0 lint issues, all three JavaScript suites, and all Go tests. `git diff --check` passed. Web App QA & Release Copilot focused Chrome 155.0.8059.40 checks passed with real HTTP against an owned ephemeral-port server using temporary local data: nine deletion/browsing groups (cancel, failed deletion, held deletion/204/trash preservation, recreate, first/last pages across years, safe previews, Back, failed-page Retry, no page exceptions). The prior eight autosave browser groups passed again. Built with `go build -o /tmp/myjournal-autosave-qa-app ./cmd/app`; browser commands were `/tmp/myjournal-csp-qa-venv/bin/python /tmp/myjournal-delete-browsing-qa.py` and `/tmp/myjournal-csp-qa-venv/bin/python /tmp/myjournal-autosave-browser-qa.py`. Evidence is in their corresponding /tmp JSON reports. Browser timers were controlled; temporary servers/browsers were stopped and data removed. No native Safari, Google, or guaranteed unload-delivery result is claimed. User server/data and private agents files were untouched; changes remain uncommitted.

## Trash icon placement (2026-10-09)

Moved the trash action into a reserved bottom-right strip inside the editor frame. The textarea ends above the strip, so scrolling/resizing cannot put text beneath the button; its native resize handle remains separate. Added the user-provided trash paths as an external assets/static/svgs/trash.svg CSS mask. The rounded-square button has a 44px target, theme-specific red, localized title/aria-label, and visible keyboard focus. Fine-pointer desktop styling softens only the background/border at rest; touch/no-hover devices retain normal contrast. Updating the deleting label now changes accessible attributes without replacing the icon. Confirmation, pending-delete locking, missing-entry hiding, and explicit-action/blur ordering remain intact.

Verification: `make fmt lint test` passed (0 lint issues, three JavaScript suites, all Go tests). Built with `go build -o /tmp/myjournal-autosave-qa-app ./cmd/app`; `/tmp/myjournal-csp-qa-venv/bin/python /tmp/myjournal-trash-ui-qa.py` passed eleven grouped Chrome 155.0.8059.40 checks on an isolated ephemeral-port local server, including existing delete/browse flows plus icon/label/mask, 44px size, geometry, hover, keyboard focus, and 390px touch emulation in both themes without horizontal overflow. Inspected desktop-light and touch-dark screenshots; all four theme/layout screenshots are in /tmp/myjournal-trash-*.png. These are Chrome emulation results, not physical-phone/native Safari evidence. Temporary server/browser/data were cleaned up. `git diff --check` passed; changes remain uncommitted and the user's autosave wording is preserved.

## Floating trash revision (2026-10-09)

User rejected the reserved footer strip. Removed that strip and restored the editor's minimum height; the rounded trash button now floats inside the bottom-right corner with an opaque theme surface background, including when disabled. Hover adds a solid red tint; keyboard focus stays visible. A modest right text inset keeps writing clear of the button without a bottom gap, and the resize handle stays exposed. This supersedes the reserved-strip description above.

`make fmt lint test` and `git diff --check` passed. Rebuilt the isolated QA binary and ran `/tmp/myjournal-csp-qa-venv/bin/python /tmp/myjournal-trash-floating-qa.py`: eleven Chrome desktop/touch-emulation groups passed, verifying no reserved footer, text clearance, opaque backgrounds, both themes, and adjacent deletion/browsing flows. Inspected the updated touch-dark screenshot. QA server/browser/data were cleaned up; changes remain uncommitted.

## Save hover highlight (2026-10-09)

Added a subtle green tint/border on enabled Save hover and keyboard focus, sharing the existing success green through --mj-success. Label contrast remains theme-native; hovering does not set Saved or change save state, and disabled/Saving buttons cannot get the highlight. `make fmt lint test` and `git diff --check` passed. Rebuilt the isolated QA binary; `/tmp/myjournal-csp-qa-venv/bin/python /tmp/myjournal-save-hover-qa.py` passed twelve grouped Chrome checks, including enabled-versus-disabled hover in both themes and unchanged status plus adjacent trash/browsing flows. Temporary server/browser/data were cleaned up; changes remain uncommitted.

## PR #8 review: uncertain deletion presentation (2026-10-09)

Verified the sole inline finding: a successful PUT adds is-saved, but failed/ambiguous DELETE previously marked recovery unverified without removing the saved class. Confirmed deletion now calls setSaveState with a neutral state before dispatch, covering pending deletion and both failure branches. Cancellation leaves the prior acknowledged presentation intact. Regression coverage starts with a real successful save in the editor harness, then exercises 503, 401, and lost acknowledgements; it checks neutral styling, retained backup/leave warning, manual retry, paused autosave, and restored saved styling after acknowledged recovery.

The new regression failed before the one-line fix and passed afterward. `make fmt lint test` passed with 0 lint issues, all three JavaScript suites, and all Go tests; `git diff --check` passed. No new browser result is claimed. Changes remain local/uncommitted; no review reply, thread resolution, push, or merge was performed.

## Calendar ordering and Settings disconnect (2026-10-09)

Calendar requests now capture the selected month and a sequence number. Older successes, delayed JSON bodies, and failures cannot replace newer results or clear their loading state. A current failure clears the grid and Today can recover. No dependency was added.

Settings disconnect confirms that saved files remain and browser drafts will be cleared. The authenticated POST /api/v1/disconnect calls the existing provider Revoke seam before clearing the session cookie and returning 204. Failures use localized JSON errors and retain the cookie. The browser shares acknowledged sign-out cleanup, broadcasts to other tabs only after success, locks overlapping account actions, and retries failed cleanup without repeating revocation. Local revocation is a no-op; actual Google revocation remains unimplemented and the Google startup guard remains intact. OpenAPI was regenerated with oapi-codegen v2.8.0.

New calendar, disconnect browser-harness, and HTTP regressions failed before implementation and passed afterward. `make fmt lint test` passed with 0 lint issues, all three JavaScript suites, and all Go tests. `go test -race ./internal/web/...` and `git diff --check` passed.

Focused Web App QA & Release Copilot checks used `go build -o /tmp/myjournal-calendar-disconnect-app ./cmd/app` and `/tmp/myjournal-csp-qa-venv/bin/python /tmp/myjournal-calendar-disconnect-qa.py`. Eight grouped Chrome 155.0.8059.40 checks passed: obsolete calendar success/error suppression, current error/Today recovery, canceled disconnect, held/failed disconnect with retained session and both tabs' drafts, acknowledged local disconnect and other-tab locking, saved-file preservation after reconnect, and no page exceptions. Settings also fit a 390px viewport without horizontal overflow. Requests used an owned ephemeral-port server and temporary synthetic local data; ordering/failures were controlled. The initial QA script expected different existing error copy; correcting that assertion allowed the run to proceed. Evidence is in /tmp/myjournal-calendar-disconnect-qa.json. Browser/server were stopped and temporary data removed. No native Safari, physical phone, or Google result is claimed. Changes remain local and uncommitted.

## Architecture enforcement and focused UI acceptance (2026-10-09)

Formatted, compile-valid forbidden imports in core, web, and local provider all incorrectly passed golangci-lint. Initial unformatted/cyclic probes were discarded as inconclusive. Depguard matches full source paths; the old relative patterns did not match, and **/*.go also failed to cover files directly in each selected directory. Corrected patterns to `**/internal/<area>/**.go` and verified both direct and nested packages. Driving-side _test.go files are exempt so existing integration tests may compose concrete adapters. Added the missing concrete clock deny. make test-architecture now checks seven violations (HTTP core, nested HTTP core, web/local, handler/clock, session/cache, local/web API, Google/web API), requires a depguard diagnostic for the selected file/import, and removes canaries on exit. It failed before the corrected patterns and passes afterward. Run separately from concurrent builds/lint because it temporarily mutates source files.

Theme detection previously wrote mj-theme on first visit, converting automatic selection into a permanent preference. A new regression reproduced that failure. The bootstrap now ignores invalid storage values and still applies device preference when storage access fails. The interactive script follows device changes until an explicit toggle and persists only that choice, preserving existing valid stored values. Existing stored automatic and explicit values cannot be distinguished, so they are both respected rather than deleting preferences. Added reduced-motion CSS, localized the main-navigation label, and made calendar month feedback a polite live status. Settings Appearance remains removed.

Verification: make test-architecture passed all seven probes; make fmt lint test passed with 0 lint issues, four JavaScript suites, and all Go packages. git diff --check passed. Built with go build -o /tmp/myjournal-acceptance-app ./cmd/app and ran /tmp/myjournal-csp-qa-venv/bin/python /tmp/myjournal-acceptance-qa.py. Seven grouped Chrome 155.0.8059.40 checks passed: device/explicit theme and keyboard toggle, keyboard Save with long Unicode/unbroken text, seven routes in each theme at 390px and CSS zoom 200% without horizontal overflow, reduced-motion transitions, editor labels/live status/disabled future date, and no page exceptions. The server used an owned ephemeral port and temporary synthetic local data; server/browser/data were cleaned up. Evidence: /tmp/myjournal-acceptance-qa.json. CSS zoom is a reflow probe, not native browser zoom acceptance; no full screen-reader/contrast certification or native Safari result is claimed.

The user requested migration analysis rather than choosing HTMX versus vanilla JavaScript. htmx-migration-assessment.md records estimated file scope, representative template/handler changes, editor safety constraints, and CSP/history-cache implications against official HTMX documentation. No HTMX library or fragments were introduced and no deviation decision was made. Theme upstream/version/license provenance remains unverified; no upstream attribution has been invented. Changes remain local and uncommitted, preserving earlier calendar/disconnect work.

## Browser architecture decision (2026-10-09)

After reviewing the HTMX migration assessment, the user explicitly chose to keep the JSON API and vanilla JavaScript and not use HTMX for this project. ADR 014 records the decision and supersedes the original HTMX requirement. The assessment retains its scope estimates, code examples, and official sources as historical research rather than an implementation plan. Updated onboarding, phase acceptance, UI checks, and progress to remove the open migration gate; theme provenance and other UI/provider acceptance remain independent. The earlier undecided assessment above describes the state before this decision.

Documentation-only update: inspected references and consistency and ran git diff --check. No runtime behavior changed and no tests were rerun for this update. Changes remain uncommitted.

## Pull request checks (2026-10-09)

Added .github/workflows/pr-checks.yml for pull_request opened, synchronize, reopened, and ready_for_review events. One Ubuntu job uses Go from go.mod, Node 24 for all four dependency-free JavaScript suites, and pinned golangci-lint v2.13.2. make fmt followed by git diff --exit-code on Go files rejects formatting drift rather than silently correcting it. make lint, make test, and make test-architecture run sequentially so temporary forbidden imports cannot overlap another build/lint. The workflow uses read-only contents permission, disables persisted checkout credentials, cancels obsolete runs per PR, and requires no app credentials or live Google tests.

Fresh make fmt lint test passed (0 lint issues, four JavaScript suites, all Go packages); make test-architecture passed all seven probes. git diff --exit-code -- '*.go' and git diff --check passed. go run github.com/rhysd/actionlint/cmd/actionlint@latest resolved v1.7.12 and validated the workflow without diagnostics; this temporary validation tool did not change go.mod/go.sum. Action versions/inputs were checked against their official repositories. No hosted GitHub run is claimed. Changes remain local and uncommitted.

## Accessibility acceptance follow-up (2026-10-10)

Started from clean main at 6e06a73 after PR #10 merge. Used the Web App QA & Release Copilot workflow with an owned ephemeral-port local server and synthetic data. axe-core 4.10.3 found dark Settings Disconnect foreground #b3261e on #1e1e1e at 2.55:1 (required 4.5:1), caused by the shared danger button selector hardcoding the light red rather than using the existing theme token. It also flagged manual review of aria-label on the generic calendar div. Reused --mj-danger for the button foreground/border and set role=group on the calendar container; no framework, provider, or save behavior changed.

Added optional scripts/python/check-accessibility.py, taking a built app, local axe script, and output report. The initial audit failed before the fixes; the final tool passes 18 WCAG A/AA-tagged scans in Chrome 155.0.8059.40 across seven routes and failed-save/conflict states in both themes. It also checks Tab reachability of all visible enabled controls and computed focus indicators on the routes/conflict editor. Arrow-glyph contrast remains scanner-incomplete; supplemental computed foreground/background ratios of 13.98–17.40:1 pass 4.5:1. Original incomplete results are retained. This does not establish screen-reader announcements, native zoom, subjective focus order, all non-text/focus contrast, native Safari, or complete accessibility compliance. Theme provenance remains open; a focused source/version question was sent to the user rather than inventing attribution.

Commands: go build -o /tmp/myjournal-acceptance-app ./cmd/app; /tmp/myjournal-csp-qa-venv/bin/python scripts/python/check-accessibility.py --app-binary /tmp/myjournal-acceptance-app --axe-script /tmp/myjournal-axe.min.js --report /tmp/myjournal-a11y-after.json. make fmt lint test passed with 0 lint issues, four JavaScript suites, and all Go packages; git diff --check passed. The axe download used the pinned npm distribution after raw repository asset URLs returned 404; no application dependency or lockfile changed. Server/browser were stopped and synthetic data removed. The QA tool remains separate from ordinary CI to avoid adding browser/tool installation to each PR run. Changes remain local and uncommitted; Google remains disabled.

## Monochrome reference review (2026-10-10)

The user identified themes.gohugo.io/themes/hugo-theme-monochrome/ as the original reference and recalled no copying. Resolved its linked upstream kaiiiz/hugo-theme-monochrome, cloned a temporary shallow snapshot at 98923c5fda5a4a6755adc6f03721f6d8db4dd060, and compared 18 local CSS/template/JavaScript/SVG files against 122 upstream source files. No identical source files, no runs of five substantive whitespace-normalized lines, no upstream mj-* vocabulary, and no exact local SVG geometry strings were found. Inspected c665672's initial app-specific tokens/base template and subsequent asset history; favicon entered separately in 9f5db7d. Upstream LICENSE identifies MIT/copyright 2021 kaiiiz. The temporary clone was removed. No code from the upstream theme was added.

This supports documenting a custom Monochrome-inspired UI, not a known source port. Updated ADR 004, onboarding, acceptance, and progress accordingly. The comparison snapshot is not a bundled version, and absence of matches does not prove that no heavily rewritten snippet or older upstream revision was ever referenced. The independently user-provided trash SVG's original source/license remains unverified; the theme comparison does not settle it. No binary favicon comparison was made. Documentation consistency and git diff --check passed; prior fresh runtime checks cover the accessibility code changes in this batch.

## OAuth browser handoff foundation (2026-10-10)

The app's provider seam previously started login with empty state, discarded Google's generated PKCE verifier, and ignored callback code/state. Replaced the string arguments with provider-neutral login request/callback values carrying state, verifier, nonce, and code. Remote login now persists independent 32-byte secrets and provider/issuance in an encrypted ten-minute `mj_oauth` cookie (HttpOnly, SameSite=Lax, configured Secure, `/api/v1/auth/`). The existing AES-GCM codec uses distinct authenticated purpose data for this cookie while preserving session encoding compatibility.

Callbacks reject missing/tampered/expired/future-dated/wrong-provider transactions, state mismatches, missing/duplicate code/state, malformed query encoding, and consent errors before calling the provider. Pending cookies are cleared; failures do not clear an existing session. Local development retains its synthetic login. Google URL construction uses the retained verifier's S256 challenge, OIDC nonce, and the actual `/api/v1/auth/google/callback` app route. Removed placeholder Google account creation and no-op revocation success: unfinished operations fail closed and the startup guard remains intact.

The transaction supports one pending login per browser. Clearing it is browser consumption, not a server-side replay ledger; copied encrypted transactions remain decryptable until expiry, and actual code exchange must enforce provider single-use codes and PKCE. No live Google exchange, signature/claim validation, token refresh, Drive storage, HTTPS/Console, or browser OAuth acceptance is claimed. Those remain the next provider work. ADR 002 and phase acceptance record these limits.

The new session test initially failed compilation because the flow API did not exist, then passed after implementation. Fresh `make fmt lint test` passed with 0 lint issues, all four JavaScript suites, and all Go packages. `make test-architecture` passed all seven forbidden-import probes. `make generate-openapi` succeeded; description-only schema edits produced no generated code changes. Handler tests verify invalid callbacks never reach exchange, correct code/verifier/nonce forwarding, cookie flags/consumption, and preservation of existing sessions on rejection. Google URL tests cover callback alignment, scopes, S256, state, nonce, and fail-closed unfinished operations. Changes remain local and uncommitted, preserving the preceding accessibility/provenance work.

Focused race checks also passed: `go test -race ./internal/session ./internal/provider/... ./internal/web/...`. `git diff --check` passed.

## Integrated Google adapter, awaiting live acceptance (2026-10-10)

Implemented code exchange with the retained PKCE verifier and the installed Google ID-token validator, then additional issuer/audience/expiry/issued-at/subject/verified-email/nonce/authorized-party checks. Require the Drive grant; reject unexpected account switching on reconnect. Compact versioned cookie credentials contain subject/refresh token only. A same-account previous connection supplies an omitted refresh token; new accounts without one fail. Access-token caches retain only access-token fields, are capped at 128 digest-keyed entries, and serialize refreshes with bounded locks. Google refresh-token rotation fails closed pending a cookie-update design. Refresh does not change session age; oversized session cookies fail before emission. Revocation uses a bounded POST and drops cache only after acknowledgement. Temporary exchange outages map to 502; invalid credentials map to 401 without provider details.

Implemented all journal Store methods with the existing Drive library. Folder roles use private app=myjournal-v1 tags, root/year identity, and deterministic oldest-created/ID selection. Renames and moved year folders survive discovery; reads do not create paths. Writes recreate missing paths, preserve duplicate folders, serialize same-account request stores, compare checksum versions, skip unchanged uploads, and verify acknowledged checksums. Listing consumes all pages and rejects incomplete/repeated-page results. Previews use four workers and at most a 64 KiB prefix, safely handling split UTF-8; files larger than 1 MiB fail full reads. Safe metadata reads retry retryable rate/transient failures at most three attempts with jitter and cancellation. Writes are never blindly retried, so ambiguous acknowledgement retains recovery/conflict behavior. Errors map to domain sentinels, not Google details.

The production registry remains guarded. GOOGLE_TEST_MODE=true permits Google only on exact localhost/loopback base URLs with OAuth client configuration; duplicate providers and remote hostnames are rejected. Added an existing-root Settings/API folder link without read-time onboarding. Regenerated Wire and OpenAPI after the force_consent query contract and callback 502 response update. User confirmed Console redirect/API/test-user prerequisites; no Console state or real account credentials were inspected. google-live-checks.md supplies startup/configuration, integrated scenarios, and a synthetic 50-changed/50-unchanged revision probe. No .env, live account data, or credentials were modified or printed.

Fresh make fmt lint test passed with zero lint issues, all four JavaScript suites, and all Go packages. Shared Store contracts pass against a paginated mocked Drive server. Additional tests cover real RSA signature/audience rejection via the installed validator, identity/grant rejection, omitted refresh/account isolation, cache bounds, revocation/refresh failures, per-request concurrent first-save convergence, rename/move/duplicate preservation, missing checksums, canceled reads, transient safe-read retry, and no ambiguous-write retry. Optional If-Match forwarding/conflict mapping is tested only against the mock. The initial auth test failed compilation before implementation; initial lint exposed unchecked closes/test encoders, which were corrected. A later duplicate import introduced during error-mapping cleanup was caught and fixed before final verification.

The local browser regression used an owned temporary-data server: 18 axe scans and keyboard checks passed with no violations or page exceptions. Evidence is /tmp/myjournal-google-local-regression.json; original incomplete contrast results remain recorded and the existing supplemental glyph checks passed. No live Google browser sign-in is claimed. Production acceptance remains blocked on real integrated OAuth/Drive checks, revision/latency measurements, external-writer/checksum-upload race policy, multi-instance policy, and remaining release/UI gates. Previous accepted spikes are not treated as integrated app evidence. Changes remain local and uncommitted and preserve earlier accessibility/provenance work.

Final verification also detected an auth handler/generated login-interface mismatch in the working tree. Reconciled the handler's consent parameter, previous-connection handoff, cookie-size guard, and temporary-outage mapping with the integrated implementation, then reran the full checks. No historical pass was used to waive that mismatch.

After reconciliation, fresh `make fmt lint test`, all seven `make test-architecture` probes, and `go test -race ./internal/provider/... ./internal/session ./internal/web/... ./cmd/app` passed. `git diff --check` passed; temporary architecture probes were removed. Live Google acceptance remains pending.

## Running-app HTTP smoke check (2026-10-10)

User requested checking localhost:8080. Direct HTTP connections to that port were refused and no TCP listener was found there; localhost:3030/healthz returned 200. On 3030, homepage/help/privacy and linked CSS/JS/favicon returned 200 with expected MIME types; security headers and no-store were present. Anonymous editor/list/calendar/settings routes redirected home, and settings API returned 401. Google login returned a redirect to Google's authorization endpoint with the 3030 app callback, S256, state, nonce, the four approved scopes, and an HttpOnly pending cookie. Redirects were not followed and credentials/cookies/query values were not printed.

Observed sign-in entry-point defect: the homepage still hardcodes Continue with local Dev and /api/v1/auth/local/login, which returns 404 in the running Google-only configuration. Direct Google login initiation works, but the homepage does not offer it. No code was changed as part of this QA request. Chrome UI inspection was unavailable: no browser automation surface was connected, then native computer use reported pending permissions and a ScreenCaptureKit capture failure on retry. No visual/console/authenticated editor or real Google consent/storage success is claimed, and no journal data was modified.

## Configured homepage sign-in controls (2026-10-10)

Fixed the observed Google-only homepage defect: the public route now derives local/Google button visibility from the injected provider registry, and the template offers only enabled providers. Google is primary when enabled; local remains primary in local-only mode and secondary when both are configured. Added the flat locale key landing.connectGoogle. Disabled-provider API requests retain their existing 404 JSON envelope.

A new route regression first failed for Google-only/both/empty registries, then passed after the change; it covers local-only, Google-only, both, and no registered providers. Fresh make fmt lint test passed with 0 lint issues, all four JavaScript suites, and all Go packages. The running 3030 homepage returned 200 and included the localized Google link without the local link. Google initiation returned 302 with the correct 3030 app callback, S256/state/nonce; the disabled local endpoint remained 404. Port availability changed during testing: an attempted make run was rejected by the existing listener, which was left untouched.

Native Chrome inspection became available on retry. Confirmed the visible Continue with Google homepage control and followed it to Google's sign-in page without provider-unavailable or redirect-uri-mismatch feedback. OAuth/account UI state was filtered before tool output; no query values, account claims, cookies, or credentials were printed. Stopped before authentication/consent and did not modify journal entries. This validates the sign-in entry point, not the full integrated Google flow. Changes remain local/uncommitted; previous work is preserved.

## Integrated Google browser smoke test (2026-10-10)

The user completed Google sign-in and explicitly authorized synthetic entry creation, edits, and trash. Native Chrome on localhost:3030 loaded a signed-in past-date editor. Verified 2026-10-01 was empty before writing; existing journal text was not changed. Manual saves acknowledged with Saved, and reload preserved pasted multiline Unicode (accented Latin, Japanese, and emoji). Entries displayed the synthetic preview; October Calendar marked the date as having an entry and disabled future dates. Settings loaded the Google connection, Disconnect control, and existing Drive folder link. The folder link's destination was present; its Drive contents were not independently inspected.

Two loaded tabs exercised a stale checksum version: after one saved a new revision, the other's save showed the expected conflict, retained its synthetic text, and offered all three resolution actions. Confirmed Chrome's unsaved-changes reload warning; the stale draft and conflict actions recovered. Keep both included local and latest saved text as Unsaved changes, then manual Save succeeded. Keep mine and Load the other version were not exercised in this integrated run.

Confirmed Move to trash acknowledged Entry moved to trash. Reload showed Blank page without the fixture or trash control. The loaded calendar removed its test-date marker, and Entries no longer showed the synthetic preview. Drive Trash contents and replacement file IDs were not independently checked; no permanent deletion was performed. Other open tabs can still display their old saved snapshots until reload.

Automation initially encountered concurrent browser interaction. An initial paste did not enter text; native typing subsequently omitted Unicode, so Unicode was retested using a verified paste and successful save/reload. These were input-automation issues, not claimed app failures. No account claims, cookie values, tokens, or existing journal text were intentionally collected. No runtime code changed during this smoke test.

This is partial live acceptance, not production enablement. Still pending: cookie inspection, restart/refresh and omitted-refresh reconnect, cancellation/grant rejection, integrated folder rename/move/recreation, the other conflict actions and external Drive writers, revocation/reconnect, Settings Disconnect, failed-network recovery, exact Drive UTF-8/LF body inspection, revision/latency measurements, and the external-writer/multi-instance concurrency policy. Existing automated and spike results do not substitute for these integrated checks.

## Integrated Google conflict and reconnect follow-up (2026-10-10)

Used an initially empty 2026-10-02 with synthetic text only. Two tabs loaded the same base; after a newer save, the stale save was rejected. Keep mine fetched the latest version and acknowledged Saved while retaining the local fixture. The other tab then reproduced another stale-save conflict. Load the other version + Cancel preserved its local draft; confirming replacement loaded the latest saved fixture and showed Saved. No network trace was collected, so the no-PUT property of that action remains supported by existing tests rather than this UI observation. Together with the earlier Keep both run, all three controls now have integrated live UI coverage.

Opening /api/v1/auth/google/login without forced consent returned directly to a signed-in editor; a subsequent read of the fixture succeeded. This proves reconnect and subsequent Drive access, but the OAuth response was deliberately not inspected: omitted-refresh-token preservation is not independently proven by this run. Requested a human-assisted restart with the same configuration/key to test fresh token acquisition without terminating the user's listener. The fixture remains saved pending that check and cleanup.

## Integrated Google restart and fixture cleanup (2026-10-10)

The user reported restarting with the same key/configuration. The first automation attempts were blocked by a locked Mac; after unlock, rebinding Chrome initially selected an unsigned incognito window. Selected the existing signed-in October 2 window through Chrome's Window menu, then explicitly reloaded the fixture. The saved synthetic text loaded with Saved status. Combined with the reported restart and the refresh-only cookie implementation, this supports successful post-restart access-token acquisition and Drive read; no token values or exchange responses were inspected. A second restart specifically after non-forced reconnect was not performed, and omission of a refresh token in that reconnect response remains unobserved.

Confirmed trash of the 2026-10-02 synthetic fixture acknowledged Entry moved to trash. Reload showed Blank page with the fixture absent. Existing journal entries were untouched. Both synthetic dates used in these smoke runs are now absent from the active journal; their recoverable Drive trash was not purged. No runtime code changed.

## Integrated external Drive revision conflict (2026-10-10)

Native setValue succeeded where automated typing/paste did not. Verified 2026-10-03 was blank, entered a synthetic fixture, saved it, and located the app-created file through Settings's Drive folder link. Drive Manage versions uploaded a generated 54-byte UTF-8/LF synthetic replacement to the existing file; upload finished and a new current version appeared. Returning to the unchanged editor showed the original saved fixture. A verified local edit followed by Save was rejected with the conflict message and local text retained. Keep both fetched the external replacement and combined it with the local draft as Unsaved changes. This verifies detection of an external edit completed before the save check; it does not establish atomic protection against a writer racing between checksum comparison and upload.

Left the combined synthetic draft unsaved for revocation recovery. Asked the user to remove the Google third-party connection and restart with the same key/configuration before retrying Save, to avoid a cached access token hiding revocation. The October 3 file and unsaved draft remain pending that check and cleanup. No existing journal text was changed; no permanent deletion or permission expansion occurred.

## Integrated revoked-access draft preservation (2026-10-10)

The user reported removing the Google third-party connection and running the app with the original October 3 editor still open. Clicked Save on the retained Keep both draft. The editor returned Please sign in again to save. Your text is still here. Draft kept locally.; both synthetic local and external text remained. This is live UI evidence of revoked-access recovery messaging and text preservation; no network status trace or token response was inspected. A transient native ScreenCaptureKit capture error occurred while reading the result; the retry succeeded.

Opened the app's forced-consent reconnect route in a separate tab and observed Google sign-in. Handed account authentication/consent to the user. Post-reconnect Save and October 3 cleanup remain pending.

### Reconnect rejection under investigation

The user reported the generic Sign-in could not be completed callback response after reconnect consent. This is a failed live reconnect, not a passing recovery check. Source inspection shows the same message covers pending transaction/callback rejection and provider authentication/claim/grant/account rejection, so the cause is not established. Asked whether Drive permission was enabled and whether completion occurred within the ten-minute flow lifetime without another login or restart/key change. Started a fresh forced-consent transaction in the reconnect tab; the original synthetic draft remains pending recovery and cleanup. No callback parameters, credentials, or cookie values were collected.

### Fresh reconnect succeeds; save acknowledgement pending

The user confirmed enabling Drive permission and completing the failed attempt within ten minutes, then reported a fresh attempt succeeded. Observed the signed-in editor and returned to the original October 3 tab: both synthetic draft versions remained with the prior sign-in-required message. Manual Save entered Saving… but repeated observations, including switching tabs and back, did not show acknowledgement or failure. Health remained 200. Do not count post-reconnect save as passed or delete the fixture until its outcome is established. Asked for human confirmation of the visible status and the query-free PUT status/duration from the app terminal. The first callback failure remains unexplained; expiry or missing consent is not established.

### Revocation/reconnect recovery confirmed and cleaned up

The user observed Saved and supplied the query-free PUT log: status 200, duration 3.8448925s. Subsequent native inspection also showed Saved. The earlier repeated Saving… observations were stale automation evidence, not a demonstrated hung request; do not treat them as an app defect. One observed request duration is not a latency distribution, and its attribution specifically to Drive was not independently measured. Reload showed both synthetic versions with Saved status, confirming persistence after reconnect.

Confirmed Move to trash acknowledged Entry moved to trash. Reload showed Blank page, no synthetic fixture, and no trash control. This completes the live sequence: external Drive revision → stale-save conflict → retained combined draft → user revocation/restart → sign-in-required failed save with retained text → fresh same-account reconnect → successful original-tab save → reload → recoverable trash cleanup. The initial reconnect callback rejection remains unexplained and recorded; the successful retry does not establish its cause. No existing journal entries were changed or permanent trash deletion performed. No runtime code changed during these checks.

## Google integration PR verification (2026-10-10)

Prepared feat/google-drive-integration with accumulated OAuth/Drive implementation, accessibility/provenance work, and integrated live evidence. Fresh make fmt lint test passed (0 lint issues, four JavaScript suites, all Go packages); make test-architecture passed all seven probes separately with cleanup. Fresh go vet ./... and go test -race -short ./... passed. git diff --check passed. Updated phase acceptance to distinguish partial live Google success from remaining release gates. Private agents/ files and .env are excluded; .env.example documents the localhost-only test switch. Production registration remains guarded.

## Go style and architecture audit (2026-10-10)

Recorded the Gophers-guided audit in [go-code-review.md](go-code-review.md). Fresh lint (0 issues), vet, race/short tests, and all seven architecture probes passed. Temporary synthetic probes reproduced invalid-date acceptance, month-13 acceptance, and failure of a year-0123 path round trip; probes and temporary data were removed. Source review also found unbounded cache generation bookkeeping, logger/config/session duplication, Settings error suppression, and rendering/cancellation improvements. No runtime implementation, live account action, commit, or push was performed for this review.

## Go review implementation (2026-10-10)

Implemented the approved review follow-up and recorded durable style/boundary conventions in AGENTS.md. See [review follow-up](go-code-review.md#implementation-follow-up--2026-10-10) for scope, regression coverage, and remaining limitations. Wire was regenerated from the composition source; generated OpenAPI was unchanged. Fresh make fmt lint test passed with zero lint issues, all four JavaScript regression suites, and all Go packages. All seven make test-architecture probes passed separately and cleaned up. Fresh go vet ./... and go test -race -short ./... passed; git diff --check passed. These are automated refactor checks, not renewed live Google acceptance. No commit, push, browser/account operation, or agents/ change was performed.

## Initial performance measurements (2026-10-10)

The isolated benchmark uses the real Drive adapter/journal service/cache and local fake HTTP, with thirty 1,920-byte synthetic entries, a 20-entry page, and six repetitions of twenty timed operations. Existing opens make 4 calls; changed saves 5; unchanged saves 4; data-cold Entries 28; warm Entries/Calendar 0; changed save followed by Entries 33. These counts validate repeated metadata discovery and broad post-save invalidation as optimization hypotheses. Local timings are not real Drive latency.

Changed saves allocate ~17.04 MB/op, compared with ~106 KB for unchanged saves. An additional whole-run allocation profile (including fixture setup) attributes ~98% of allocation space to the installed Google SDK's media buffer; its default upload chunk is 16 MiB. This is evidence for a separately measured upload-buffer candidate, not a latency improvement or a production heap estimate. See [benchmark protocol/results](performance-benchmark.md) for raw repetitions, source manifest, profiling caveats, and live-probe instructions. No optimization was applied.

Fresh make fmt lint test passed (zero lint issues, five JavaScript suites, all Go packages). Eight benchmark scenarios passed a race-detector smoke with one iteration each. All seven architecture probes passed separately with cleanup; git diff --check passed. Chrome app availability was observed, but the browser automation read-only scope cannot execute the authenticated timing probe or access Performance entries. Live timing collection was handed to the user without exporting session material; no live entries were created by automation. Existing AGENTS.md semantic-format edit is preserved.

## Live exploratory performance result (2026-10-10)

User supplied a successful browser probe with sixty HTTP-200 timed requests and cleanupVerified=true. Independently recomputed medians: open 2.270 s; changed save 3.333 s; unchanged save 1.815 s; repeated Entries 4.55 ms (first sample 7.074 s); repeated Calendar 4.95 ms; Entries after changed save 7.078 s, excluding the preceding save. All ten post-save browsing samples took 6.601–7.842 seconds. This supports invalidation/repopulation as a user-visible bottleneck, consistent with synthetic counts, but live request/cache/retry attribution remains uninstrumented. Calendar was measured after Entries had populated indexes. Upload allocation is a separate proven local issue, not a demonstrated cause of live latency. No optimization applied.

Raw attachment reports January 1; accompanying user message reports January 2. Stored the sanitized raw result unchanged and flagged the discrepancy in its manifest; requested clarification without another run. The user also reported subsequent January 1 preflight refusals (zero samples), demonstrating no overwrite. Historical cleanup success does not establish that date's current state or the cause of any retained fixture. No additional account mutation or cleanup was attempted. See [performance results](performance-benchmark.md#live-exploratory-results--2026-10-10). Documentation/data changes only; git diff --check passed.

The user subsequently confirmed the attachment was the object displayed after the January 2 invocation. Its raw date remains January 1. The probe returns its date argument directly; no cause for the discrepancy is established. Both reports are retained in the manifest without relabeling samples or inferring current fixture state. Timing conclusions remain exploratory and unchanged.

## Partial benchmark failure and safe diagnostics (2026-10-10)

New attachment labels January 2 and corroborates earlier timings (post-save browse median 6.673 s), reporting sixty HTTP-200 samples and cleanup verified. Console history includes an earlier pending run and results arriving after subsequent invocations; this supports asynchronous completion/possible overlap as the explanation for date-label confusion, without proving exact concurrency. Preserve individual run dates and do not pool samples.

January 3 stopped after ten successful opens, nine successful changed saves, then HTTP 500 at 21.743 s. Retained all twenty measured samples including the failure; cleanupVerified=false and upload outcome unknown. No account mutation/cleanup or automated retry was attempted. The external-client timeout is a hypothesis, not a proven live cause. Requested safe terminal duration and public error message. A separate wrapped-deadline regression reproduced HTTP 500 fallthrough; changed classification to existing HTTP 503 unavailable response. Probe now reports dated scenario progress and rejects overlapping invocations; tests cover guard release after setup failure. Fresh make fmt lint test passes, with zero lint issues and five JavaScript suites. No performance optimization applied.

January 3 response follow-up: user supplied HTTP 500 with the generic JSON message “Something went wrong. Please try again.” and response Date 2026-10-10 03:11:02 UTC (10:11:02 WIB). This matches the handler fallback envelope, but supplies no backend error category or terminal duration. The accompanying JavaScript stack identifies the request/save call site only. Deadline fallthrough remains a reproduced code defect and a plausible explanation for this run, not a confirmed live root cause. Copied request headers, payload, and base version were not persisted; only sanitized response metadata was added to the failed-run manifest. No request was replayed or fixture changed.

## Measured performance candidates (2026-10-10)

Implemented the approved candidates sequentially, with six twenty-operation repetitions per stage. Small changed-save allocation fell from median ~17.04 MB/op to ~0.17 MB/op using SDK ChunkSize(0); single-request uploads retain size/checksum checks and no ambiguous retry. Bounded operation-local exact-query reuse reduces cold Entries from 28 to 24 HTTP calls. A domain-owned optional operation-store port keeps Google details outside the core, and cache decorators forward the fresh view.

Targeted invalidation retains unrelated snapshots without extending expiry, drops all affected-date preview budgets (including unchanged acknowledgements), independently checks year/all-years membership, and conservatively invalidates indexes on create/trash/ambiguous failure. The scalar epoch still rejects stale racing fills, with no account-generation map or acknowledgement-content seeding. Off-page save/page drops from original 33 calls to 5; newly added visible-date comparison drops from matching scoped baseline 29 to 9. Local timing medians are descriptive, not real Drive speedups or statistical significance; benchstat is unavailable. Raw outputs/manifests and policy tradeoffs are in the performance report and ADR 015.

Fresh make fmt lint test passes (zero lint issues, five JavaScript suites, all Go packages), all seven architecture canaries pass separately, and go vet passes. The browser probe now supports selected scenarios with safety/selection regressions. Requested a same-config app restart and targeted live result; the January 3 fixture remains untouched, and no new live fixture was created by automation. Private plan updated locally; no commit or push.

Final verification after independent year-list membership refinement: fresh go test -race -short ./... passes for all packages, as do go vet ./..., make fmt lint test, and seven separate architecture probes. Final candidate source hashes include production boundaries, fake transport/fixtures, benchmark harness and regression tests. Targeted live comparison remains pending; no live speedup is claimed.

## Targeted live candidate comparison (2026-10-10)

The user completed the January 4 targeted probe: ten samples each for changed saves, repeated Entries, and post-save Entries, all HTTP 200; cleanupVerified=true. Against January 2, post-save browse median fell from 6.673 to 2.277 seconds (~66% lower). Repeated Entries median was 3.70 ms, with its first request taking 3.756 seconds. Changed-save median rose from 3.385 to 3.632 seconds (~7.3%); no live save-speed improvement is established.

Keep the three bounded candidates based on synthetic allocation/request reductions, passing automated checks and exploratory live browsing evidence. Retained sanitized samples/manifest in benchmarks/results/2026-10-10-live-targeted-candidate; details in performance-benchmark.md. Different dates, sequential runs, unverified exact build/cache state, earlier possible overlap and absent live request/retry traces limit attribution. Thirty successful samples do not establish failure-rate acceptance; January 3 remains untouched and unexplained. No new runtime changes, commit, push or production enabling accompany this evidence update.

## Save timing investigation prepared (2026-10-10)

Source/synthetic evidence still shows five upstream calls for changed saves; live timing has not attributed their latency. Added debug-only Drive transport timing with bounded categories/status/outcomes, header time and body completion time; OAuth acquisition is included in header time and body consumption includes decoder time. No raw errors, URLs, IDs, payloads or account values are emitted. Concurrent operations cannot be correlated from these records; documented idle-tab sequential capture. Added browser-only two-GET January 3 inspector comparing exact known synthetic A/B bodies without returning content/checksums or writing/deleting. Requested user-run flags; January 3 remains untouched and its historical write outcome/cause unconfirmed. No retry, timeout increase or new optimization.

Fresh verification: make fmt lint test passed with zero lint issues, five JavaScript suites and all Go packages; go test -race ./internal/provider/google and git diff --check passed.

## January 3 read-only inspection follow-up — 2026-10-10

The user ran the browser inspection helper and reported exists=true, stableAcrossReads=true, matchesLastAttempt=true, matchesPreviousAcknowledged=false, matchesInitialFixture=false, cleanupPerformed=false. The [sanitized flags](../../benchmarks/results/2026-10-10-live-jan03-failed/inspection.json) establish that the exact final attempted B content is currently present across two fresh reads despite the earlier HTTP 500. This supports treating the failed response as an ambiguous acknowledgement rather than proof that the write was rejected. It does not establish historical commit timing or identify the failing backend operation; the original timeout hypothesis remains unconfirmed. No retry, save or cleanup was performed. Live debug timing collection remains pending.

## Live save timing breakdown — January 5 fixture (2026-10-10)

The user supplied ten successful changed-save samples (browser median 3.469 s, range 3.329–4.483 s) and debug logs; cleanupVerified=true. Retained [sanitized samples and grouped timings](../../benchmarks/results/2026-10-10-live-save-timings/raw.json) exclude machine hostnames, request IDs and raw log lines. The first PUT creates the fixture and is excluded; the subsequent ten PUTs align sequentially with browser samples. Every measured save contains three lists, one fresh metadata request and one upload, all HTTP 200 with complete bodies. There are no extra attempts visible in this capture.

| Stage | Median total per save | Share of aggregate server time |
| --- | ---: | ---: |
| Three discovery lists | 1.360 s | 40.5% |
| Fresh metadata/conflict check | 0.342 s | 9.9% |
| Single upload | 1.665 s | 49.5% |

Median server duration is 3.464 s. Timed upstream requests account for 99.94% of aggregate server duration; median remainder is 1.89 ms. Independent stage medians need not sum to the overall median. The maximum header-to-body-completion gap is only 1.67 ms in these measured requests, so response consumption is not the observed bottleneck. This is time within the wrapped external call, including potential token acquisition/transport work, not proof of time spent inside Google's servers.

Sample four's 4.479 s server duration includes a 2.757 s upload. Sample ten's 4.285 s includes 2.309 s of discovery (one list takes 1.432 s), while its upload takes 1.653 s. Both discovery and upload variability matter; this run does not support a single failing stage or explain January 3. No failures/retries were observed, which does not prove future reliability.

Decision: do not optimize local CPU, increase timeouts, remove the fresh conflict check or add automatic write retries from this evidence. Upload allocation changes improved memory rather than eliminating upstream upload latency. Further latency work would need a separately reviewed discovery-request reduction or bounded hint-cache experiment preserving external rename/move/trash/duplicate behavior and fresh pre-upload validation. Persistent hints remain deferred; no new caching policy is implemented here. First retain this baseline and address ambiguous-write recovery/concurrency acceptance before expanding caching. Exact build identity and per-request correlation remain unavailable; sequential idle-window grouping is the attribution assumption.

User decision: discontinue January 3 root-cause investigation. Existing evidence remains recorded; this is a deferral, not a confirmed fix or cleanup authorization. Continue with general ambiguous-save recovery and external-writer concurrency acceptance.

## Explicit retry reconciles uncertain saves (2026-10-10)

Approved bounded design implemented: manual Save after a non-conflict failed save first reads the current entry with cache:no-store. Exact retry-snapshot matches acknowledge current content/version without another upload; valid absence acknowledges an empty snapshot. Different text retains the original base for the conditional PUT, preserving conflict controls. Failed/malformed/unauthorized reads make no PUT, preserve drafts and keep autosave paused. Later typing remains editable/backed up, and sign-out prevents late writes. Initial and ordinary automatic saves gain no extra GET. Reload uses its existing initial-read reconciliation. No claim of historical outcome, atomicity or external-writer protection; ADR 005's production gate remains open. January 3 root-cause work stays deferred and its fixture is untouched.

New dependency-free editor regressions failed before implementation and pass afterward, covering committed-but-unacknowledged saves (503/network/truncated JSON), matching reads without PUT, original-version retry/conflict, later typing, failed verification/401, malformed versions, empty/absent text, sign-out races and reload reconciliation. Added browser acceptance instructions; no new human/browser acceptance result claimed.

Fresh verification for uncertain-save retry: make fmt lint test passes with zero lint issues, all five JavaScript suites and all Go packages. git diff --check passes. No commit or push performed.

## Uncertain-save local browser validation (2026-10-10)

Agent-driven Chrome/macOS QA against localhost:3030 with local provider passed the recovery checklist using a disposable February 20, 2020 entry. Verified lost acknowledgement preserves text/backup and pauses retry; matching retry trace PUT→GET has no duplicate upload; newer retry trace PUT→GET→PUT conflicts rather than silently rebasing. Held verification keeps typing enabled and later input Unsaved changes, with subsequent save success. Rejected verification GET retains backup without a PUT; explicit retry recovers. Separate-tab sign-out during held GET prevents late PUT/backup recreation. Restored native fetch and local login, trashed the disposable fixture through confirmation, and reloaded to Blank page. January 3 remains untouched.

Evidence is recorded in editor-recovery-checks.md. Network failure/delay were scoped DevTools fetch injections, not a browser-wide Offline toggle. Exact Chrome version was not captured; Safari and Google are not covered. An unsuccessful initial probe-paste attempt is excluded. No product code changes or new runtime test run were needed for this QA evidence update; git diff --check passes. Workflow used the Web App QA & Release Copilot skill.

Performance plan maintenance (2026-10-10): refreshed stale opening/completion checkboxes in the private plan and tracked implementation plan. Preserved original proposal as historical, recorded actual evidence limits, and removed January 3 investigation as active work. Private agents files remain outside commit scope. Documentation-only update; git diff --check passes.

## Shared test fixture review (2026-10-10)

Reviewed user-added internal/testutil constants with Gophers Go review/declaration guidance. A dependency-free shared fixture is appropriate across test packages; renamed MockBaseUrl to LocalBaseURL with URL initialism casing and documentation. Reused the origin in request/provider setup, while preserving literal default/redirect expectations and loopback security boundary cases to avoid tautological contract tests. Production code does not import testutil; live HTTP tests retain httptest.Server URLs. Renamed the local provider callback constant to CallbackPath and retained an independent full redirect expectation. Corrected the incidental README API prefix edit back to the implemented /api/v1. Repeated strings alone do not justify a global constant: extract values only when they share fixture/policy meaning.

Final PR preparation verification after shared test fixtures: fresh make fmt lint test, all seven separate architecture probes, go vet ./... and go test -race -short ./... passed. Private agents/ files and environment/local data are excluded from the commit.

### Empty-entry status correction (2026-10-10)

The editor input handler previously labeled any clean text as Saved, ignoring whether the loaded entry existed. It now uses the acknowledged existence flag to distinguish Blank page from Saved. Interaction regressions cover absent and existing empty entries, typed-then-erased drafts, and unchanged-content autosave suppression. The regression failed before the fix; make fmt lint test passes afterward.

## Google reliability regression coverage (2026-10-10)

Reviewed the clean main branch and tracked acceptance gates. Added focused tests for the existing bounded safe-read retry policy and applied-but-unacknowledged uploads; no runtime behavior or dependencies changed. testing/synctest exercises cancellation/deadline during backoff deterministically. The tests distinguish retryable rate limits/server failures from permanent authentication/permission/not-found/precondition failures and unclassified transport errors, preserve original error identity, and enforce the three-call bound.

Mocked Drive create/update scenarios apply the upload before returning 503. The adapter reports unavailable without retrying the mutation; a fresh Get observes the applied synthetic content and checksum, and retrying the original base conflicts without another upload. This confirms why failure cannot be interpreted as an unapplied write. It complements browser retry tests; it does not establish external-writer atomicity or live reliability. No real Drive data or app session was accessed. Fresh make fmt lint test, go test -race ./internal/provider/google, and git diff --check pass.

## Restricted Google release preparation (2026-10-10)

The user accepted one active app process, no simultaneous external edits, and documented residual lost-update risk for a first restricted release (ADR 016). Existing checksum and process locks are unchanged; production enabling still awaits live and deployment acceptance. Reviewed official Drive update/performance guidance without establishing atomic conditional-upload support.

Live session began through Web App QA & Release Copilot: localhost initially refused connection, then signed-in Google Settings and folder link were visible after user startup. The initially blank 2019-02-20 page accepted disposable synthetic text and DOM status reached Saved. Settings Disconnect confirmation appeared, but automation timed out accepting it; human completion and same-account reconnect were requested. No Disconnect/reconnect pass is claimed yet. Fixture cleanup remains pending; no real journal entries or existing folders were altered.

Google Disconnect follow-up: user confirmed that accepting Settings Disconnect returned to the signed-out page. Subsequent Google login returned the generic sign-in rejection. This establishes user-observed sign-out, not a complete Disconnect/reconnect or saved-file-preservation pass. Requested only the sanitized sign-in rejection stage and callback HTTP status to distinguish browser transaction, exchange, claims, grant, account and refresh rejection. No root cause established; synthetic 2019-02-20 fixture retained.

### Live Google Disconnect/reconnect (2026-10-10)

User confirmed Settings Disconnect returned to the signed-out page. Reconnect initially failed with stage=grant and HTTP 401; reported granted scopes omitted drive.file, and the user confirmed the Drive permission had not been explicitly selected. After explicit Drive consent, the user reported successful login. Agent reloaded the saved synthetic 2019-02-20 entry and observed its original text and Saved status, establishing file preservation across Disconnect/reconnect. The pre-reload editor showed disabled Save/trash and sign-in feedback following sign-out. This rejection is expected fail-closed behavior, not an unexplained callback failure; it does not explain the earlier unrelated callback rejection. No credentials or callback queries are retained here.

### Live disposable-year recreation (2026-10-10)

Agent inspected the app-created 2019 folder: its only active file was the known synthetic 2019-02-20 entry. Trashed that file through Drive, verified the folder empty, then trashed the empty year through Drive. Reloading the app date returned Blank page; refreshing the parent Drive folder showed no 2019 folder, confirming that this read did not recreate it. Saving new synthetic text reached Saved and recreated a 2019 folder containing the dated text file; app reload returned the new text and Saved. Trashed the recreated file through Drive and verified app reload returned Blank page. The empty replacement year remains; original trash is retained, no permanent deletion, and root/entries/other years were untouched. This covers integrated year recreation, not root/entries recreation or a new app-confirmed trash check.

Revision measurement handoff: Privacy tab prepared; user asked to run only the changed stage of the existing probe on 2019-02-21 before Drive-version inspection and the unchanged stage. Results and fixture cleanup remain pending. No runtime code changed in this live QA step; git diff --check passes.

### Live changed-save revisions (2026-10-10)

User ran the existing revision probe on initially absent synthetic date 2019-02-21: changed stage completed 50 saves, 50 checksum changes, elapsedMs=228968. Supplied request logs show successful uploads/PUT acknowledgements; no raw logs, machine names, request IDs or account data are retained here. Total probe duration includes deliberate pauses and browser overhead and is not a per-request latency metric.

Agent inspected Drive Manage versions after completion: Current version plus numbered Versions 1–49, totaling 50 displayed versions. No coalescing was observed in this single run. This is a point-in-time observation, not a revision-retention guarantee; Drive warns temporary versions can be removed automatically. There was no pre-existing fixture/version baseline. Unchanged stage and cleanup remain pending; user asked to run the unchanged stage once before the next inspection.

Unchanged revision stage interrupted (2026-10-10): user supplied a PUT for synthetic 2019-02-21 returning HTTP 500 after 1.496 seconds. No completed probe result or failing iteration was available, so the 50 unchanged-save gate remains open. Cause and whether any preceding requests succeeded are unestablished; retain the fixture and do not infer version changes from this failure. The automation-created Privacy tab was not retained across the previous turn, which may explain its closure independently of the API failure. Request only sanitized upstream category/status/outcome timings before retrying.

Follow-up sanitized timings locate the failed unchanged PUT before mutation: two Drive list requests completed with HTTP 200, then a third list ended with status=0/outcome=transport_error after approximately 412 ms; the app returned HTTP 500 after 1.496 seconds. The preceding unchanged PUT was acknowledged with HTTP 200 after 1.922 seconds. There is no upload in the supplied failed-request sequence. This supports a pre-upload discovery transport failure, not an ambiguous write acknowledgement. The underlying transport error (for example cancellation versus network failure) is not exposed by these timings and remains unknown. Unknown transport errors are intentionally not blindly retried. A fresh unchanged-only probe may proceed after the previous probe has stopped, retaining the changed-stage fixture and rechecking its expected content.

### Live unchanged-save revisions (2026-10-10)

Fresh unchanged-only run completed: user reported 50 saves, 0 checksum changes, elapsedMs=120301. Supplied logs show all 50 PUTs acknowledged with HTTP 200, with list/metadata requests and no uploads during the stage. Agent reopened Drive Manage versions after completion and counted Current version plus Versions 1–49, the same 50 versions observed after the changed stage. Unchanged writes therefore produced no additional displayed revisions in this run. Total elapsed time includes deliberate pauses/browser overhead. The earlier interrupted transport failure remains recorded separately; the successful rerun does not establish its root cause or a retention guarantee. Synthetic fixture cleanup is still pending.

Revision fixture cleanup (2026-10-10): user confirmed trashing synthetic 2019-02-21 through the app and reloading to Blank page. Changed/unchanged revision observation and recoverable cleanup are complete; production acceptance remains separate.

### Dependency vulnerability baseline (2026-10-10)

Ran `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` using Go 1.27.1. Scan failed (scanner exit status 3, surfaced by go run as exit 1), reporting 12 reachable standard-library advisories, fixed in Go 1.27.2; some also identify golang.org/x/net v0.58.0 with fixes in v0.60.0. Additional imported-package/module findings require review. Reachability is scanner evidence, not proof each advisory is exploitable in this deployment. Production gate remains open. The declared Go version is preserved pending the explicit project-version decision; CI already reads go.mod. Official example: <https://pkg.go.dev/vuln/GO-2026-6617>.

Live refresh capability after sign-in/restart (2026-10-10): user completed app Google login without forced consent using the same account, restarted with unchanged key/configuration, and confirmed existing entries 2026-10-08 and 2020-01-03 loaded. This establishes working persisted refresh capability after that flow, but does not prove Google omitted a refresh token in its OAuth response; no token values were inspected or exported.

Dependency remediation (2026-10-10, user authorized): upgraded declared Go to 1.27.2 and golang.org/x/net to v0.60.0; module resolution also advanced x/crypto to v0.57.0, x/sys to v0.48.0 and x/text to v0.42.0. Ran go mod tidy. govulncheck v1.8.0 with -show verbose exits successfully: zero reachable vulnerabilities and zero imported-package findings. One module-only advisory remains, GO-2026-5932 for unmaintained x/crypto/openpgp, which is not imported; no fixed version exists. Do not describe the entire dependency graph as advisory-free. Existing golangci-lint v2.13.2 cannot decode Go 1.27.2 export data, including when rebuilt with that toolchain; v2.14.0 passes, so CI pin updated. Fresh make fmt, make test (all five JavaScript suites and Go packages), make lint with v2.14.0 (zero issues), and sequential make test-architecture (all seven probes) pass. Production HTTPS/deployment/privacy/live recovery gates remain open.

Privacy audit follow-up (2026-10-10, partial static inspection): request logging omits queries/headers/bodies and Google timing logs use bounded categories without raw errors/URLs. However NewRouter still installs chi middleware.Recoverer, whose pinned implementation prints the recovered panic value and backtrace to stderr. This can disclose sensitive values carried in a panic and is inconsistent with the project logging requirement. No actual disclosure or attacker-triggerable panic is established. Replace with sanitized panic recovery and regression coverage before marking the privacy/panic gate complete. Audit remains partial; startup/config and server error paths still require review.

### Google-backed lost-acknowledgement recovery (2026-10-10)

User ran the scoped synthetic response-loss wrapper on disposable 2019-02-22. Reported Save failed. Draft kept locally.; manual Save retry issued a GET with no additional PUT and reached Saved. This establishes Google-backed current-content reconciliation after a deliberately hidden response, not a real upstream timeout cause or atomic external-writer guarantee. No text/cookies were collected. The five-second no-automatic-retry observation was not separately reported. Later-typing, verification failure, sign-out race and fixture cleanup remain separate live cases.

Google uncertain-save newer-text follow-up (2026-10-10, human reported): after synthetic response loss, further typing displayed Unsaved changes with automatic saving paused. Manual Save performed GET then a conditional PUT returning HTTP 409; latest text remained and conflict controls appeared. Keep mine subsequently returned a successful PUT and Saved. This validates retaining the original conflict base when current Drive content differs from newer local text; no blind version adoption occurred in this case. Delayed verification, failed verification, sign-out race and cleanup remain pending.

Google uncertain-save verification follow-ups (2026-10-10, human reported): typing during a throttled recovery GET passed the requested preservation/status check. In a separate offline verification attempt, user observed no PUT, only the entry request; after restoring online access, manual Save succeeded. These establish the reported delayed-read and failed-read recovery outcomes, not an upstream Google outage. The greater-than-five-second no-automatic-retry observation was not separately confirmed. Sign-out during pending verification and fixture cleanup remain open.

Google sign-out during recovery verification (2026-10-10, human reported): original editor displayed Please sign in again to save. Your text is still here.; user observed only GET, with no follow-up PUT. After signing in again, edited text remained and the status was Saved. This records user-visible recovery and absence of a late mutation; browser-storage cleanup/no-recreation was not independently inspected, and the Saved observation alone does not establish a new upload or historical outcome. Synthetic 2019-02-22 cleanup and explicit waiting-period confirmation remain pending.

Google recovery sequence completion (2026-10-10, human reported): user confirmed no automatic retries after failed-save feedback and trashed disposable 2019-02-22, then reloaded to Blank page. Together with preceding reports, the scoped Google response-loss/reconciliation sequence and fixture cleanup are complete. Browser-storage deletion was not independently inspected; synthetic response loss does not establish real outage causes or atomic external-writer safety.

### Sanitized request panic recovery (2026-10-10)

Replaced chi default Recoverer with focused web middleware logging only a generic request panic event and request ID; recovered values/backtraces are not formatted or logged. Router supplies the localized generic API JSON envelope or browser text error. http.ErrAbortHandler is rethrown without recovery logging. A panic after committed headers logs the sanitized event and aborts rather than appending a misleading error body. Regressions cover synthetic sensitive panic data omission, generic API JSON/500 integration, abort propagation and committed-response behavior. Fresh make fmt lint test passes (zero lint issues, all five JavaScript suites and Go packages); git diff --check passes. This closes the identified chi panic-log issue, not the complete startup/server/privacy audit.

### HTTP server diagnostic privacy (2026-10-10)

Static follow-up found http.Server.ErrorLog unset, allowing net/http to print raw diagnostic/panic values and peer addresses outside router recovery. Added a generic server log writer which discards supplied diagnostic bytes and emits only HTTP server error through the configured slog default. Wired ErrorLog in wire.go and regenerated wire_gen.go; inspected generated changes. A regression injects synthetic sensitive diagnostic data and verifies only the generic event is emitted. The tradeoff is intentionally reduced low-level diagnostic detail. Startup review found configuration validation returns fixed messages without credential values; startup logs use validated base URL, port and provider IDs, while listener/shutdown errors remain operational errors. Embedded template/locale initialization can panic on build-owned assets, not runtime journal content. Request logs intentionally retain query-free paths (including entry dates), status/duration and request IDs; Google logs retain bounded categories. No production log sink/proxy retention audit was performed. Fresh make fmt lint test passes; git diff --check passes.

Railway preparation (2026-10-10): user selected <https://myjournal-ink.up.railway.app>. Published GitHub Releases remain the trigger. Prepared pinned CLI upload from an exact git archive, deployment-ID status polling and public health verification. No actual deployment is claimed. Initial public GET /healthz returned 404; Google composition-root registration remains localhost-only and public acceptance is pending.

Railway preparation validation: fresh make fmt lint test passed (0 lint issues), all seven architecture probes passed, actionlint passed both workflows, three offline deployment-status regressions passed and git diff --check passed. Official Railway JSON schema was downloaded, but local jsonschema tooling was unavailable; schema validation and authenticated deployment remain pending.

Railway acceptance (2026-10-10): user reports runtime variables and exact Google callback configured. Fresh public GET /healthz returned HTTP 502, superseding the earlier 404 observation; this does not establish a failure cause. Current source still rejects non-test Google registration. Requested a generic deployment startup diagnostic without secret/config values before attributing the failure. Public OAuth acceptance remains pending.

Railway startup diagnosis (2026-10-10): user provided the generic startup rejection from the current main build, confirming the Google live-acceptance guard blocks startup. Local acceptance change replaces that unconditional rejection with HTTPS-origin and Secure-cookie requirements while preserving loopback-only test mode. Focused registration regression failed against the old guard, then passed after the change and Wire regeneration; make fmt lint test passed with 0 issues. No public deployment or OAuth acceptance is claimed; automation readiness remains unset.

Public Railway HTTPS checks (2026-10-10): user reports deployed server with HTTPS base URL and Google registered. Independent public GETs to /healthz, / and /privacy returned 200; unsigned entry GET returned 401. Responses include CSP, no-store, no-referrer, nosniff and DENY framing; HSTS was absent. A separate unsigned login request returned 302 with a ten-minute transaction cookie marked Secure/HttpOnly/SameSite=Lax. Invalid dummy callback returned 401 and cross-origin PUT returned 403. Chrome landing visibly exposes Continue with Google. Dummy non-secret query marker sent for human platform-log review; no real consent, authenticated session-cookie inspection, Drive write or restart/refresh is claimed. Endpoint checks are not full OAuth acceptance.

Railway logging/browser follow-up (2026-10-10): user supplied application logs show query-free callback path at the dummy-probe time, with stage=transaction/401 as expected; the dummy query marker is absent in supplied logs. This covers the visible application logs, not independently inspected HTTP logs/exports/internal retention. Clicking Continue with Google in the acceptance Chrome tab reached Railway HTTP 502 (Application failed to respond); no real consent occurred. Hosting availability must recover before public OAuth acceptance.

Railway OAuth retry (2026-10-10): fresh independent /healthz returned 200 and login returned 302. Browser retry reached Google authorization error 401 invalid_client, with OAuth client not found. This is an authorization-stage rejection before callback/code exchange; deployed client ID verification is requested without collecting its value. No consent, authenticated session or Drive smoke acceptance is claimed.

Railway client-ID correction (2026-10-10): user reports verified/corrected configuration deployed. Fresh browser login reaches Google account chooser and no longer shows client-not-found. Human consent is pending; this confirms authorization initiation, not completed OAuth or Drive access.

Railway authenticated smoke (2026-10-10): human confirmed completed Google sign-in and Drive consent. Chrome opened empty 2019-02-23 (Blank page), saved a disposable synthetic Unicode fixture, observed Saved, then reloaded and observed the identical fixture with Saved rather than draft-restoration feedback. This establishes public authenticated save/read behavior. Restart/refresh, authenticated session-cookie attributes and cleanup remain pending; no existing journal content was modified.

Railway restart/cookie acceptance (2026-10-10): user reported service restarted; subsequent Chrome reload remained authenticated and returned the identical synthetic 2019-02-23 fixture with Saved. Given the implemented refresh-token-only session and process-local access-token cache, this exercises fresh token acquisition after the reported restart; no token values were inspected. Human DevTools inspection reports mj_session Secure/HttpOnly/SameSite=Lax/Path=/ and expiry 2026-11-09, consistent with the fixed 30-day policy. Synthetic trash confirmation appeared, but automation timed out during acceptance and subsequent observation; cleanup is not yet verified.

Railway fixture cleanup (2026-10-10, human reported): user accepted the pending trash confirmation and reloaded 2019-02-23 to Blank page. This closes cleanup of the public HTTPS synthetic fixture; no permanent deletion was performed.

Railway navigation/sign-out smoke (2026-10-10, browser observed): calendar completed its October 2026 grid (42 cells), entries completed Page 1 without Retry, and Settings Sign out returned the signed-out landing. Navigating directly to the synthetic entry route afterward redirected to /. No journal previews or account details were included in evidence. This verifies ordinary public sign-out, not provider revocation; public Disconnect/reconnect remains pending.

Railway Disconnect/reconnect (2026-10-10, human reported): after the automation tab closed, user opened a normal Chrome tab, performed Settings Disconnect, then signed in again using the same account and explicit Drive permission. The app returned to /entries/2026-10-10. This establishes the reported public disconnect/reconnect journey; remote token invalidation and saved-file preservation across this particular disconnect were not independently inspected. No new fixture was created.

Release-readiness verification (2026-10-10): fresh govulncheck v1.8.0 ./... and verbose follow-up report zero called-symbol and zero imported-package vulnerabilities. Module-only GO-2026-5932 concerns unimported golang.org/x/crypto/openpgp (no fixed version); this is not reported as an application vulnerability. Three deployment-helper unittest cases pass. actionlint is unavailable on PATH; Docker CLI exists but its daemon is unavailable, so no fresh workflow-lint or local container smoke is claimed. HSTS design approval and external Console/infrastructure/release configuration evidence are pending.

HSTS/release configuration (2026-10-10): approved one-year host-only HSTS middleware is registered when the startup-configured base URL has HTTPS scheme; client-supplied forwarded protocol cannot enable or disable it. Regression covers landing, static, missing and protected API responses for HTTPS and HTTP localhost; it failed against absent HSTS before implementation. Fresh make fmt lint test passes (zero lint issues and all five JS suites). Workflow now reads RAILWAY_PROJECT_ID from secrets to match human-reported repository configuration; repository RAILWAY_TOKEN remains supported, with environment-scoped secrets preferred. Human confirmed one replica and disabled branch autodeploys, Console audience Testing, and observed callback HTTP logs without query strings. No export/internal-retention, deployed HSTS, rollback or release-triggered deployment claim is made. Google Testing restrictions remain applicable, including seven-day refresh-token expiry for Drive access ([Google OAuth documentation](https://developers.google.com/identity/protocols/oauth2)).

HSTS verification follow-up: fresh make test-architecture passed all seven probes after implementation; git diff --check passed. actionlint and local Docker smoke remain unavailable as noted above. User edits in AGENTS.md and README.md were preserved.

Restricted release decision/logging follow-up (2026-10-10, human reported): user selected restricted test-user releases, accepting seven-day refresh-token expiry and measured save-latency/revision-retention limitations, and confirmed no Railway log drain is enabled. This closes audience/launch decisions and configured external-drain review. It does not establish undocumented internal provider retention, unrestricted Console publication, deployed HSTS or rollback evidence.

Deployed HSTS verification (2026-10-10): after human-confirmed deployment of merged changes, independent unsigned HTTPS requests observed /healthz=200, landing=200, theme-init.js=200, settings API=401 and missing route=404. Every response includes Strict-Transport-Security: max-age=31536000 and CSP; non-static responses include no-store. No authenticated content or cookies were collected. Rollback/session continuity and current-build restoration remain pending before automated release readiness.

Railway rollback/restoration (2026-10-10): human reports rollback to the PR #15 deployment, health and existing entry access without another sign-in, followed by redeployment of latest commit. Independent unsigned /healthz, /privacy and /help returned 200 with Strict-Transport-Security: max-age=31536000 after restoration. Combined with the accepted restricted test-user audience and preceding checks, this satisfies the recorded activation prerequisites; the readiness repository variable was not accessed or changed, and the first actual published-release workflow remains unverified.

Public-page header regression (2026-10-10): user screenshots show Settings gear absent on Privacy/Help while authenticated. Source routes omitted SignedIn. Added regression for valid, expired, malformed and missing cookies on both public routes; valid-session cases failed before the fix. Routes now use existing session validation solely for header state and remain publicly accessible. No provider or Drive call is introduced. Local changes are not deployed.

Privacy/Help header fix validation: fresh make fmt lint test passed with zero lint issues, all five JavaScript suites and all Go packages; git diff --check passed.

First release automation (2026-10-10): connector inspection of [job 114229543859](https://github.com/mridha-work/myjournal-app/actions/runs/38057735570/job/114229543859) shows every step successful. Logs identify tag v0.1.0 at commit 8ae5dc70a74e3e45d59dd4da659b6efec24b8431, zero lint issues, no called-code vulnerabilities, and deployment 171322d7-df21-45fb-a273-895c7a38ad4c succeeded for that uploaded source with health passed. Independent public health/Privacy/Help requests return 200, one-year HSTS and CSP. Signed-in Chrome Privacy and Help visibly include Settings, closing the reported gear regression. This validates the first release workflow and restricted test-user readiness; it does not publish Google Console or prove atomic external-writer safety. Existing local workflow variable edit was preserved.

Progress reconciliation (2026-10-10): audited all 21 unchecked PROGRESS.md rows against later tracked evidence. Marked 19 superseded completion/policy gates complete, converted one optional local preview item to explicit deferral, and retained one full-UI acceptance umbrella covering native zoom/screen-reader, remaining focus/non-text contrast and external icon provenance. Revised stale blocker/next-step summaries. Completed checkboxes retain limitations for non-atomic writes, exploratory performance, local Docker availability and visible-versus-internal platform logging. This documentation audit adds no new runtime verification.

README/AGENTS reconciliation (2026-10-10): compared docs with committed generators/Makefile, Google startup policy, HSTS middleware, release workflow and tracked v0.1.0 evidence. Updated stale pending-release language and development-only feature summary, documented restricted audience/accepted limitations and current ID/token placement, retained open full UI acceptance, and clarified fresh checkouts use committed generated files. Public-page Settings and Google Disconnect evidence now match current behavior. Local links and git diff --check pass; runtime tests were not rerun for documentation-only changes. No commit/push performed.

README scope decision (2026-10-10): user requested restoring make generate despite committed generated files and keeping current development status out of README. Restored generation with tool prerequisites and generated-diff review, replaced dated status with durable features/usage constraints, removed first-release history from release instructions, and recorded the convention in AGENTS. This supersedes the preceding quick-start recommendation.

## Markdown formatting checks (2026-10-10)

Checked all 42 tracked Markdown files with markdownlint-cli2 v0.23.2
(markdownlint v0.41.1). Fixed bare URLs, redundant blank lines, table formatting
and an unescaped path example. The repository configuration preserves existing
long-line prose by disabling MD013; all other default rules remain enabled.
Private `agents/` files are excluded. Tooling ran from a temporary directory,
without adding a project npm dependency or installing a skill.

## Duplicate entries loading message (2026-10-10)

The Entries template supplied a list placeholder while JavaScript populated a
separate live status with the same message. Moved initial loading feedback into
that existing status region and left the list empty until data arrives. Added
rendered-template and pending-fetch regression coverage. `make fmt lint test`
passed (0 lint issues, all browser suites and Go tests). `git diff --check` and
Markdown lint passed. No deployed browser check was performed.

## BigCache storage adapter (2026-10-11)

Replaced the custom production cache map/list with installed BigCache v1.2.1.
`cachestore` owns the byte-storage port and journal policy; the concrete adapter
lives in `internal/adapter/cache/bigcache`. Wire initialization now returns cache
cleanup through BuildApp, which retains its existing request-draining lifecycle.

Fresh checks: `make generate-wire` passed and its initialization/error/cleanup
diff was inspected. Focused tests for cachestore, BigCache, Google synthetic
storage and cmd/app passed. `make fmt lint test` passed with zero lint issues,
all five JavaScript success markers and all Go packages. Focused race tests passed.
An initial lint run caught an unchecked test cleanup error, which was corrected
before the successful full run. No human/live/production acceptance is inferred.

The adapter bounds byte queues to 16 MiB across sixteen shards, not total process
memory or exactly 2048 entries. It rejects oversized entries before writes and
checks cancellation during lock waits and prefix deletion. The decorator enforces
absolute expiry independently of library cleanup and retains the accepted
invalidation/generation policy. Unexpected cache errors disable reuse for the
coordinator; backing reads/writes continue with their original results. ADR 017
records eviction changes, prefix-scan cost and future distributed-coordination
requirements. Existing progress and installed dependency edits were preserved.

The separate `make test-architecture` run passed all fourteen forbidden-import
probes, including nested cache-policy and concrete-adapter boundaries. Probe files
and the temporary nested directory were removed. A final author review checked
the approved design, generated Wire diff, failure behavior and dependency direction;
no independent reviewer was delegated, following the user's inline execution choice.

Final formatting check: `git diff --check` passed; documentation paths and
consistency were reviewed. Markdown lint could not be rerun: `command -v
markdownlint-cli2` returned 1, and the attempted `npx --no-install markdownlint-cli2`
invocation returned 127 (`npx` is unavailable). No tools were installed for this.

## Retain the previous LRU as an optional adapter (2026-10-11)

The user revised the earlier single-adapter scope to retain the previous LRU
implementation as backup/historical code. Restored its map/list storage approach
in `internal/adapter/cache/lru`, adapted to the existing consumer-owned byte port.
The default remains 2048 entries, with an explicit capacity override, LRU promotion
on reads/updates and lazy exact-expiry rejection. Byte ownership, prefix deletion
and context-aware coordination match the port. Journal policy is shared rather
than duplicated in this adapter. Its limit bounds entry count, not total bytes;
it owns no background work and needs no lifecycle cleanup.

Wire and generated composition continue to select BigCache. Switching adapters
would require an explicit composition-root change; there is no automatic fallback
or new environment setting. ADR 017 and architecture guidance record the revised
scope. The existing depguard patterns already cover the LRU directory, and a new
temporary adapter-to-web probe verifies that boundary.

Tests first failed because the new adapter constructors did not exist, then passed
with the implementation. Fresh `make fmt lint test` passed with zero lint issues,
all five JavaScript success markers and all Go packages. Shared journal store
contracts passed through both adapters. `go test -race
./internal/adapter/cache/lru ./internal/cachestore` passed. The separate
`make test-architecture` passed all fifteen probes and removed temporary files.
Author review confirmed the old storage semantics, current port contract and
unchanged BigCache selection. Documentation paths and `git diff --check` passed;
Markdown tooling remains unavailable as recorded above. No commit or deployment.


## Hexagonal boundaries and application lifecycle (2026-10-11)

The approved refactor groups domain/use cases under internal/core, HTTP under
internal/driver/web, and provider/cache/codec/clock adapters under internal/driven.
Connection workflows consume a lifecycle Gateway/Catalog port, independently of
storage construction. Core owns session/login models, fixed lifetimes, binding,
consent/reconnect eligibility, issuance, disconnect and settings. HTTP retains
cookie metadata, encryption adaptation, query multiplicity/size checks, redirects
and safe response mapping. Google signature/claim/refresh behavior is unchanged.
Synthetic ciphertext generated before model extraction still decodes with the
same fields and authenticated-purpose isolation. Revocation failures retain the
session; both HTML/API settings use the shared application workflow.

Wire constructs renderer/catalog with initialization-error propagation and narrow
web options; cache cleanup remains intact on initialization errors and shutdown.
BigCache remains selected at driven/cache/local/bigcache; LRU remains optional.
The journal decorator owns ByteCache with unchanged byte copying, cancellation,
miss/capacity and prefix-deletion contracts. No remote adapter or runtime selector
was added. Shared journal contracts continue to run against local, mocked Google
and both cache adapters. Journal service/editor pages share pure date validation,
including year-zero rejection. Local operational filesystem failures now wrap
ErrUnavailable with preserved identities; metadata failures/disappearance abort
the list rather than presenting partial success. Local-origin validation is shared
neutral connection policy rather than an adapter dependency on startup config.

Fresh verification: make generate passed using oapi-codegen v2.8.0 and installed
Wire; generated initialization/cleanup and API changes were inspected. Go 1.27.2
and golangci-lint v2.14.0 run make fmt lint test successfully, with zero lint issues,
all five JavaScriptCore success markers and all Go packages passing. The separate
make test-architecture run passed thirty-one forbidden-import probes and removed
all temporary source/directories. The new startup-config probe initially hit an
import cycle through local tests; moving it to a nested package proved depguard
rejection instead. Cleanup was verified on that failure too. The pre-refactor
nested core-to-config probe was accepted by lint, establishing the original gap.

`go test -race ./internal/core/... ./internal/driven/... ./internal/driver/web/...
./cmd/app` passed. Local Markdown links, active source/generator references and
git diff --check passed. `command -v markdownlint-cli2` returned 1, so no fresh
Markdown-lint pass is claimed. Inline review checked dependency ownership, cookie
compatibility, error identities, date boundaries, cache coordination and generated
cleanup. No reviewer was delegated and no agent commit, push or deployment occurred.
The main migration was committed externally as 5948fd3 during the work; follow-up
changes remain in the working tree. ADR 018 records the durable decisions.

This adds automated structural/contract evidence, not live Google, performance,
accessibility or deployment acceptance. Existing restricted-release status,
stateless callback replay limits and non-atomic external-writer/deployment-overlap
risks remain unchanged.


Final review follow-up: a focused regression exposed rename failures using
os.LinkError rather than os.PathError. The adapter now maps both to ErrUnavailable
while preserving underlying identity; cancellation/not-found/conflict/invalid-text
identities remain unchanged. The regression failed before the correction and the
fresh full make fmt lint test run passed afterward (zero lint issues, all browser
success markers and Go packages). The focused local race check was repeated.


## Import grouping enforcement (2026-10-11)

The reported import block was in internal/driver/web/handler/auth_test.go; the
Google adapter test already grouped imports correctly. gofmt alone preserved
incorrect group boundaries. Enabled bundled gci with standard/default/project
prefix sections in .golangci.yml, and changed make fmt to golangci-lint fmt ./....
No new dependency or tool installation was needed. The configured lint check
first rejected existing grouping drift, then passed after repository-wide
formatting. Project aliases stay in the project group and sort by import path.
Generated code is excluded from formatter rewrites and remains generator-owned.
AGENTS.md records the convention and repeatable command.

Fresh make fmt lint test passes with zero lint issues, all five browser success
markers and all Go packages. golangci-lint fmt --diff ./... returns no differences;
git diff --check passes. Generated Wire/OpenAPI files have no follow-up diff.
Existing working-tree edits were preserved. No commit or push was performed.

The separate make test-architecture run also passed all thirty-one probes after
formatter configuration; all temporary probes were removed.

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

## Public sitemap (2026-10-12)

Implemented public /sitemap.xml and /robots.txt in the web delivery layer, using configured APP_BASE_URL. Sitemap lists only /, /help, /privacy and /terms. Focused route tests cover valid XML, exact URL set, content types and untrusted Host isolation. make fmt lint test passed; Wire regenerated. After deployment, submit sitemap.xml in the matching Google Search Console property. No deployment or submission performed.

## Security review remediation: development isolation (2026-10-12)

Local login intentionally issues a credential-free dev identity, so checking APP_BASE_URL alone did not protect the wildcard socket. The composition root now binds development configurations to loopback, preserving IPv6 loopback when configured and production Google networking. S01/S02 historical spikes also bind loopback and use a bounded, expiring shared spike-only transaction store (128 entries, ten minutes); callbacks consume once and login capacity failures return 429. Wire regenerated. Fresh make fmt lint test and targeted race checks (go test -race ./cmd/app ./spike/...) passed. This is source/regression-test evidence, not a new live Google or LAN acceptance run.

Sequential make test-architecture passed all 31 probes and cleaned up temporary files; git diff --check passed.

## Sitemap record structure corrected after PR review (2026-10-12)

PR 23 identified well-formed XML with an invalid sitemap record layout: all locations shared one url element. Now each public location has its own url record. A structural regression failed before the serializer change and passes afterward; fresh make fmt lint test and git diff --check passed. Earlier XML/URL checks established syntax and content only, not correct sitemap record cardinality.

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

scripts/python/check-accessibility.py now expects the Settings gear in both session states, adds signed-out Settings to both themes, checks that account actions are absent there, and scans the open header language menu. Fresh local execution passed 34 axe scans with zero violations, 30 keyboard-focus checks and zero page errors. Report: /private/tmp/myjournal-header-a11y-20261012.json. Axe retains incomplete aria-controls checks and sampled contrast checks; direct DOM assertions confirmed each controlled ID is unique and references a listbox, open menus render below their trigger, and computed header-language-button contrast exceeds 4.5:1 in both themes. The existing arrow contrast assertions also passed. These checks do not establish native screen-reader announcements or full WCAG conformance. Historical “no signed-in Settings link” coverage above is superseded by the user-approved always-visible public Settings navigation.

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

### Shared button hover border (2026-10-12)

Help and Today now share the theme-text border accent used by header/icon controls. The shared enabled-button hover/focus rule also covers the Settings language trigger and trash; Save uses the same border while retaining its green action background, and trash retains its danger tint. Disabled/aria-disabled controls are excluded from the new accent, and generic button lift now excludes them. Fresh make fmt lint test and git diff --check passed. No browser visual or accessibility scan was rerun. The user subsequently authorized committing this change and opening a PR for the current branch.

## Overlapping storage work (2026-10-12)

The existing operation-scoped discovery and TTL caches did not collapse concurrent cold misses. The shared journal cache decorator now uses singleflight for public Store reads, benefiting both current providers and future adapters wired through it. Provider/account, generation and read parameters isolate flights; successful work is shared across devices with different grants. Save/trash attempts separate subsequent flights; mutations are never collapsed. Completed editor content is never persisted by the coordinator. Google-specific discovery remains operation-local. [ADR 019](adr/019-overlapping-journal-reads.md) records bounds, cancellation behavior and tradeoffs.

Synthetic eight-request Entries coverage reduced downloads from 160 to 20 and metadata lists from 32 to 8; calendars reduced metadata lists from 24 to 3. Different Store operations no longer share individual Google folder queries, so the earlier uncommitted Google-specific 4-list result is superseded. Existing TTL staleness, deployment overlap and external-edit risks remain. See [Progress](PROGRESS.md#bounded-overlapping-journal-reads-2026-10-12) and the [performance report](performance-benchmark.md#overlapping-drive-reads-2026-10-12) for fresh checks and measurement limits. No live latency improvement or unrestricted release acceptance is asserted.

Provider/account sharing refinement: credential fingerprints and `WrapScoped` were removed by user decision so different laptops/grants share successful reads. An authentication failure does not automatically fail followers: each can verify once through its own Store, under the same coordination admission bound. Both-expired credentials return authentication errors without a retry loop. Fresh full formatting/lint/tests and focused race checks passed; no live multi-device Google measurement was performed.

## Recent Entries and Go boundary triage (2026-10-12)

Confirmed early termination in renamed `Service.ListEntries`: it retains size+1 dates, stops older-year traversal once that lookahead is filled, and previews only returned dates. Complete per-year metadata still loads for each visited year. The browser now shows only the latest twenty and routes older-note browsing to Calendar; legacy URL query parameters cannot request later pages through this UI. The API retains pagination and validation.

The handler-owned `JournalService` interface replaces the concrete factory return; a direct stub test proves isolated handler use. Existing preview concurrency remains bounded at four; additional year prefetch/worker increases need measurement. [The review](go-concurrency-interface-review.md) records other interface candidates without introducing unused abstractions. Fresh full checks and focused race tests passed. Chrome verified the twenty-entry limit with 25 synthetic entries and both locale behavior; 34 axe scans passed after fixing the new Calendar link's color-only distinction. These are local/synthetic checks, not Drive latency or native screen-reader acceptance.

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

Fresh `make fmt lint test`, cache race checks and `git diff --check` passed at this checkpoint; no deployment or live Drive measurement was performed.

Changing only the rendering constant from twenty to ten required no production JavaScript or translation changes: both already consume the rendered value. JavaScript fixtures were aligned to ten and retain an independent five-entry case; earlier twenty-entry benchmarks remain valid evidence for their original workload.

Pre-PR review found no blocking issue. Fresh `make fmt lint test`, `go vet ./...`, `go test -race -short ./...`, all 31 architecture probes and `git diff --check` passed. The reviewed branch includes bounded shared reads, operation-local discovery reuse, ten preview workers and the configurable ten-entry browser view. Live Drive speed/throttling and cross-process guarantees remain unverified.

## Browser sign-in failure recovery (2026-10-12)

Cancelled Google consent, rejected required grants and other callback failures accepting HTML now redirect with 303 to `/sign-in-failed`. The EN/ID recovery page explains cancellation, missing permissions and expiry, describes the Drive requirement and provides Back to home for retry. Existing connections and browser drafts remain intact; the pending OAuth cookie is cleared for registered remote-provider callbacks. State/PKCE/nonce checks and provider token-grant verification remain unchanged. No arbitrary callback errors, codes or state values are rendered or forwarded. JSON clients retain existing error envelopes/statuses; the OpenAPI description/303 response is updated and generator output was checked (no generated diff).

The page uses shared typography/theme/navigation, permits language switching back to itself, sends no-store/noindex headers and stays out of the sitemap. Accessibility QA now includes it for signed-in and signed-out contexts in both themes. Fresh `make fmt lint test`, web race checks, and 38 Chrome axe scans plus keyboard checks passed. Automated coverage includes denied consent without exchange, rejected grants without session creation, pending-cookie clearing, preserved session cookies, query-free redirects, JSON fallback, public rendering and both locales. Earlier provider tests continue to verify missing Drive grants are rejected. Axe incomplete checks remain human review items; this is not full accessibility certification. No live Google consent flow or deployment was performed.

All 31 architecture probes and final `git diff --check` also passed; temporary QA server/data/binary and architecture probes were cleaned up. Changes remain uncommitted.

## Entry cache validation (2026-10-13)

A fixed-clock regression verifies that a second ordinary read avoids underlying Get, retains text/version together, and expires exactly at two minutes without sliding expiry. An explicit fresh read sees an external change without replacing/renewing the ordinary snapshot. Successful save refreshes the next read while preserving an unrelated date's snapshot; successful trash removes cached content, and missing entries remain uncached. A separate account performs its own read. A controlled in-flight old read cannot refill after save invalidation.

Core tests verify fresh-port selection, uncached-provider fallback and future-date validation before I/O. Handler tests verify ordinary versus Cache-Control: no-store selection and no-store response headers. JavaScript regressions verify fresh requests for recovered drafts, failed-save verification and conflict actions. Existing account-isolated singleflight/cancellation/admission and mutation tests continue to pass.

Fresh `make fmt lint test`, focused core/cache/Google/web race checks and 38 Chrome axe scans plus keyboard-focus checks passed. OpenAPI regeneration completed without a generated-code diff. These checks use synthetic/local fixtures; no live Drive latency, external-writer atomicity or deployment-overlap guarantees were established. Automated accessibility retains incomplete checks. The established conservative invalidation after failed/ambiguous mutation attempts is retained alongside the requested successful-mutation invalidation.

Sequential architecture validation passed all 31 forbidden-import probes and cleaned up temporary files; final `git diff --check` passed. Owned accessibility QA process/data/binary were cleaned up.

## Confirmed absence cache validation (2026-10-13)

A missing-entry snapshot records confirmed absence explicitly and returns ErrNotFound on hits, preserving the existing HTTP 200/exists:false mapping. Two repeated absent reads make one backing call. The fixed-clock test confirms expiry exactly at two minutes; an externally created file remains hidden to ordinary hits until expiry but is immediately visible through GetFresh. Creation via Put invalidates cached absence; Trash invalidates cached content, and its subsequent confirmed absence can be reused. Availability, authentication and cancellation errors each trigger another backing call rather than becoming blank-page snapshots. This supersedes the earlier uncached-absence observation above.

Fresh `make fmt lint test`, focused cache/core/web race checks and `git diff --check` passed. OpenAPI regeneration completed without generated-code changes. No live Drive latency measurement or deployment was performed.
