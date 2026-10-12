# Agent guide for myjournal-app

## Start here

This guide contains durable instructions and user decisions. All required context is in this repository; private planning files and historical chat exports are not prerequisites. Inspect affected code before treating a requirement or historical passing check as current behavior.

Read the user's current instructions first, then consult:

- [Progress](docs/development/PROGRESS.md) and [findings](docs/development/findings.md) for status and evidence.
- [Phase acceptance](docs/development/phase-acceptance.md) for detailed requirements and remaining gates.
- [ADRs](docs/development/adr/) for architecture decisions and [Phase 1 review](docs/development/phase-1-review.md) for Google evidence/fallbacks.
- [OpenAPI](docs/api/openapi.yml), affected code/tests, and the [recovery checklist](docs/development/editor-recovery-checks.md) as relevant.

## Stack, setup, and checks

Use Go, chi, `html/template`, embedded assets, Google Wire, and oapi-codegen. Preserve module `github/mridha/myjournal-app` and the declared Go version (currently 1.27.2). Prefer the standard library and existing dependencies.

```sh
# Create .env from .env.example; set SESSION_KEY using openssl rand -base64 32.
make run               # loads .env; go run ./cmd/app does not
make fmt lint test     # required for implementation changes
make test-js           # only browser interaction regressions
make test-architecture # temporary forbidden-import probes; run without concurrent lint/build
make generate-openapi  # after OpenAPI edits; requires oapi-codegen
make generate-wire     # after Wire edits; requires wire
```

- Defaults: port 3030, `APP_BASE_URL=http://localhost:3030`, `PROVIDERS=local`, `LOCAL_DATA_DIR=./.devdata`, info logging, insecure cookies on HTTP localhost. Keep listener port and base URL aligned; they are separate settings.
- `SESSION_KEY` is base64-encoded 32 random bytes. Startup rejects invalid keys, unknown/duplicate providers, and local mode outside exact localhost/loopback hostnames. Never print `.env`.
- Edit `cmd/app/wire.go`, regenerate `cmd/app/wire_gen.go`, and version both. Edit `docs/api/openapi.yml`, regenerate `internal/driver/web/api/openapi.gen.go`. Never hand-edit generated code or restore top-level generated directories.
- The schema declares OpenAPI 3.2.1 and the generated header identifies oapi-codegen v2.8.0. Verify installed compatibility; do not silently downgrade. `make generate` runs both generators; `go generate ./cmd/app` also runs Wire. Inspect generated diffs.
- `make test` runs JavaScript regressions and `go test ./...`. JavaScript requires Node.js or macOS JavaScriptCore, without npm packages. Preserve the JavaScriptCore success-marker check so asynchronous assertions cannot appear to pass.
- `make fmt` runs the installed `golangci-lint fmt ./...` with gofmt and gci. Group imports as standard library, third-party packages, then `github/mridha/myjournal-app` packages, separated by blank lines and sorted alphabetically by import path within each group (including aliased imports). Apply this to all handwritten Go files, including tests, Wire inputs and spikes. `make lint` rejects drift. Generated code stays generator-owned and is excluded from formatter rewrites; regenerate it with the owning tool instead.
- Lint uses golangci-lint v2.14.0 (v2.13.2 cannot read Go 1.27.2 export data). For documentation-only changes, check links/paths, consistency, and `git diff --check`; runtime tests are unnecessary. Report unavailable tools and exact commands/results; never present historical checks as a fresh run.
- Spikes also use port 3030. Do not kill an existing listener without establishing ownership.

## Product constraints

`myjournal` is a quiet app for one plain-text entry per day. Production is a stateless bridge to the user's Google Drive; local storage is development-only.

- Files are UTF-8 plain text with LF endings at `myjournal/entries/<year>/YYYY-MM-DD.txt`. No JSON wrapper, body metadata, or rich-text rendering.
- Request only `openid`, `email`, `profile`, and `https://www.googleapis.com/auth/drive.file`. Never broaden scopes to fix visibility.
- Past dates remain editable without a cutoff; future dates are rejected. Browser IANA time zone defines today; a selected date never shifts at midnight.
- Delete means trash, with confirmation. Reads never create folders; writes recreate missing paths. Discover folders by private `appProperties`, not names. Preserve renamed and duplicate/data-bearing folders.
- Connections belong in encrypted, authenticated HttpOnly cookies. No production user database or server-side session store; bounded in-memory caches are allowed. Keep Redis deferred.
- v1 is online-only, with English and Bahasa Indonesia interfaces. Language selection in Settings persists in a browser preference cookie after sign-out; default to English. Browser backup is recovery, not offline sync. No additional providers, attachments, Markdown, client-side encryption, search, tags, sharing, reminders, AI summaries, native apps, or journal-content analytics. Park ideas in `docs/development/ideas.md`.

## Architecture

| Path | Responsibility |
| --- | --- |
| `cmd/app/` | Configuration, provider registration, Wire composition root |
| `internal/core/journal/`, `internal/core/journal/storetest/` | Domain/use cases, Store/Clock ports, shared contracts |
| `internal/core/connection/` | Connection/session models, lifecycle policy, ports, registry and application services |
| `internal/driven/provider/{local,google}/` | Development filesystem and Google OAuth/Drive adapters |
| `internal/driven/cache/journal/` | Journal cache decorator and consumer-owned byte-cache port |
| `internal/driven/cache/local/{bigcache,lru}/`, `internal/driven/clock/` | BigCache storage (production), optional historical LRU storage, and system clock adapters |
| `internal/driven/session/`, `internal/config/` | AES-GCM cookie codec and environment validation |
| `internal/driver/web/`, `internal/driver/web/handler/`, `internal/driver/web/middleware/` | Router/pages, thin API handlers, focused middleware |
| `internal/driver/web/api/openapi.gen.go` | Committed generated API code |
| `internal/driver/web/i18n/`, `assets/` | Locale catalog, embedded templates/static files |
| `tests/`, `spike/` | Browser regressions and standalone Google experiments |

- Dependencies point inward. `.golangci.yml` enforces recursive core, driver, driven, cache-policy, codec and startup-configuration boundaries; only driving/driven test files may compose concrete adapters. Domain code must not import HTTP/chi, rendering, OAuth, Google APIs, filesystem, or database packages. Providers must not import web.
- `journal.Service` owns open/save/delete/list/calendar. Handlers parse requests and call the injected session-scoped service factory. Only `cmd/app` wires concrete providers, cache adapters, and system clocks into production web/session code. `connection.Service` owns login, completion, disconnect and settings; HTTP owns cookie adaptation and response mapping. Session models/lifetimes belong to core, encryption to driven/session. Cache policy consumes `cachestore.ByteCache`; concrete cache storage belongs in `internal/driven/cache/local/` (ADR 017).
- Keep Store/Clock in `internal/core/journal/store.go`; use `journal.Clock` and `internal/driven/clock.System`, not a new utilities abstraction. Store methods are context-aware and `journal.Version` is opaque. Map wrapped errors with `errors.Is`; do not expose provider details to generic code.
- Preserve shared account-isolated caches, invalidation/generation guards, bounded memory, page-only previews, and Get bypass. Pagination validates pages 1–10000/sizes 1–100 before storage access and retains at most size+1 dates.
- Optional operation-scoped discovery reuse must start fresh for each journal service call and stay bounded; cache decorators forward the scope without changing shared cache ownership (ADR 015). Keep downloads and pre-upload checks fresh. Targeted invalidation preserves unrelated snapshots only until their original expiry, always invalidates the affected preview, and retains generation guards; never seed caches from potentially out-of-order save acknowledgements.
- Run shared store contracts for every adapter. Use fixed clocks, temporary directories, and mocked external services. Depguard's production file patterns are verified by make test-architecture (thirty-one temporary violations including nested imports). Driving-side test files may compose real adapters for integration tests; production files may not. Keep this check separate from concurrent builds/lint and preserve cleanup.
- APIs use `/api/v{version_number}/...`; `/healthz` is unversioned. Browser routes are separate. Use JSON and vanilla JavaScript by explicit user decision (ADR 014). HTMX is excluded from this project; the original HTMX plan is superseded. The migration assessment is historical research, not a backlog. Do not add fragment routes or silently introduce a frontend framework.

## UI and localization

- Preserve Monochrome-inspired responsive styling and licenses. Hugo is not a runtime/build dependency. ADR 004 records the source comparison: Monochrome is a visual reference, with no copied theme code detected; do not describe an upstream theme version as vendored. Separate external icon provenance remains distinct.
- Keep the initial editor empty, without sample content or placeholder. Use readable typography, plain text, preserved line breaks, and wrapping for long text/emoji.
- Header: brand, centered Write/Entries/Calendar, theme toggle, globe-and-EN/ID pill language picker, always-visible Settings gear, and Help. Privacy remains linked from Help. Settings and language preferences are public; validate the session before showing account data, Sign out or Disconnect. Keep narrow-screen spacing/wrapping. Footer keeps `Built for quiet writing.` independently centered, with the build version and fixed 2026 GitHub copyright at the right on wide screens; narrow screens stack the details on a centered second line. Release tags are embedded at build time from the deployment archive’s RELEASE_VERSION; local builds omit the version.
- Appearance was removed from Settings by user decision. Do not restore System/Light/Dark controls there; the shared header owns switching. Without a saved preference, follow device-theme changes; persist only explicit toggle choices. Existing valid preferences remain respected.
- Icons belong in `assets/static/svgs/`, not inline HTML. Use theme-aware currentColor masks and accessible labels/tooltips. Keep the evenly spaced gear and PNG favicon under `assets/static/favicons/`.
- Use theme variables, visible keyboard focus, live-region status, disabled semantics, adequate touch targets, WCAG AA contrast, reduced-motion support, and 200% zoom. Color alone must not communicate status/calendar state.
- Save feedback says `Saved`, not `Saved locally`; editing returns it to neutral grey. Today's next-date control is disabled; invalid/future routes use the styled 404. Previews start at the first meaningful line after leading whitespace.
- `assets/locales/en.json` stays flat with dotted keys; no nesting/flattening. Templates use `t`, scripts use `data-messages`. All new labels/errors/accessibility copy belong there; dates, account labels, and user text are runtime data.
- Escape user text through templates/safe DOM text APIs. Never trust journal content as HTML. No third-party page requests, fonts, trackers, or CDN assets.

## Text safety and date invariants

Preserve these accepted behaviors when implementing autosave or other features:

- Account/date-scoped `mj-draft:v3` backups use sessionStorage, independently per tab. Reload/same-tab navigation recover them. The user accepted expiry when a tab closes; fresh-tab recovery is not required.
- Draft scope excludes account labels/credentials. Older local v2/date-only backups require explicit recovery; retain known base versions and remove the old copy only after successful backup migration.
- Empty/whitespace edits are drafts. Never clear text on failed loads/saves, 401, conflicts, or ambiguous acknowledgements. Storage errors must not falsely claim backup success; advise keeping the page open/copying text.
- Initial HTML is read-only and Save disabled until a valid GET. Failed load offers Retry. Typing stays enabled during PUT; only acknowledged snapshots become clean. Later edits remain unsaved using the returned version.
- Manual retry after a non-conflict failed save first reads fresh current content. Exact snapshot matches can acknowledge without another PUT; differing text keeps the original conditional version. Failed verification retains the draft and paused autosave. Preserve later typing and sign-out guards; this reconciles current content without proving historical write outcome or external-writer atomicity (ADR 013).
- Stale recovered drafts retain conflict actions. Keep mine fetches latest then conditionally saves; Load the other version confirms replacement and makes no PUT; Keep both appends saved text below local text for review before Save. Further changes can conflict again.
- Logout must succeed before draft deletion/final cross-tab notification. Pending/failed logout preserves recovery and editor usability. Post-logout cleanup failure gets a truthful signed-out message and cleanup-only retry. Preserve other accounts/theme; see ADR 011 for notification/storage limits.
- Confirmed entry trash locks mutations until acknowledgement; clear only this tab/date backup after 204, retain text and mark it unverified on failed/ambiguous DELETE, and prevent blur autosave from preempting explicit actions. List pagination uses bounded page links and same-page Retry.
- Resolve URL-encoded IANA `mj_tz` once per request, defaulting missing/invalid/oversized/Local to UTC; keep embedded tzdata. Pages, calendar, and future checks share the resolved location and injected clock.
- Correct stale timezone cookies before editing, without reload loops when storage is blocked. Canonicalize default editor routes to an explicit date; reload preserves it. A loaded page's today marker is a snapshot until navigation/reload.
- Autosave uses five-second idle debounce, unchanged-content suppression, one request in flight, and follow-up saves against acknowledged versions. Failures pause automatic writes until manual success or explicit conflict resolution. Recovered/migrated drafts and Keep both output wait for review (editing or Save); blur/hidden/page-hide flushing is best effort and never clears unacknowledged backups. Preserve ADR 013 and the autosave regression checklist.

## Google and current priorities

- Restricted Google test-user release acceptance is complete; the first published-release workflow v0.1.0 passed on 2026-10-10. Google Console remains Testing; do not describe this as unrestricted public publication. ADR 016 records the user-approved restricted first-release policy: one Railway service at <https://myjournal-ink.up.railway.app>, accepted temporary process/revision overlap, no simultaneous external editing, and documented residual lost-update/duplicate-entry risk. The single-instance topology is best effort across deployments; do not claim atomic concurrency or deliberately scale to multiple instances. GOOGLE_TEST_MODE=true permits the integrated OAuth/Drive adapter only on exact localhost/loopback URLs; non-test Google registration requires an HTTPS origin and Secure cookies; do not weaken those startup checks. S1/S2/S4/S7 and the Phase 1 go decision are recorded; consult tracked evidence instead of routinely asking to repeat accepted experiments.
- Use Drive md5Checksum for content versions (ADR 005), while treating pre-upload comparison as non-atomic and handling absent checksums explicitly. Share folder locks per account; select duplicates deterministically without deleting them. Single-process convergence does not prove multi-instance safety.
- Preserve existing refresh tokens when later OAuth responses omit them. The approved lifetime is fixed 30 days from original IssuedAt (ADR 002); refresh must not extend it. Server age and cookie expiry are enforced; a shorter future limit must apply to existing cookies too.
- Spikes use `/auth/google/callback`; app auth uses `/api/v1/auth/{provider}/...`. The app now persists state/PKCE/nonce in a separate ten-minute encrypted login cookie and validates the browser callback before exchange; see ADR 002 for single-pending-login and stateless replay limits. Code exchange, signature/claim/nonce checks, account-bound refresh preservation, revocation, and Drive storage are implemented with synthetic tests; use google-live-checks.md for scoped human-assisted regression checks. Integrated and public HTTPS acceptance evidence is recorded; do not routinely repeat it or claim actual refresh-token omission was observed in the integrated live flow. Console publishing/verification remains separate.
- Local atomic saves, same-process mutation protection across stores, repeated trash preservation, and UTF-8 validation/previews are implemented. Focused Chrome UI acceptance now records native 200% zoom, keyboard/error-state focus, sampled non-text contrast, human-reported screen-reader announcements and SVG Repo CC0 trash-icon provenance (ui-acceptance-checks.md); retain its coverage limits. Preserve the accepted restricted-release status without marking full Phase 2/3 UI acceptance complete.
- Use the adopted net/http.CrossOriginProtection direction for unsafe browser requests; SameSite alone is insufficient, and OAuth still needs state/PKCE. Session age, a 1 MiB save-envelope cap, and path-only logging are implemented. HTTP deadlines/draining and same-origin CSP/security headers are implemented (ADR 012). HSTS uses a one-year host-only policy for the configured HTTPS origin, without subdomains/preload; never derive this policy from untrusted forwarded headers. Preserve external scripts and verify affected browser behavior. API errors/input/cookie documentation are aligned; preserve their JSON envelopes, explicit save strings, and wrapped domain error mappings.
- Published non-prerelease GitHub Releases with vX.Y.Z tags with optional version suffixes deploy through .github/workflows/release.yml; tag pushes alone do not. Preserve main-ancestry/code checks, serialized deployments, source archives, deployment-ID SUCCESS polling and public health verification. Keep one Railway replica and branch autodeploys disabled; inspect ambiguous deployments before retrying.
- Release configuration: repository variables RAILWAY_RELEASE_READY=true and RAILWAY_PROJECT_ID; production-environment variables RAILWAY_SERVICE_ID/RAILWAY_ENVIRONMENT_ID; secret RAILWAY_TOKEN (environment-scoped preferred, repository secret supported). Never turn the token into a plain variable. Release publishing still requires task authorization.
- The user accepted Google Testing restrictions, normally seven-day refresh-token expiry for Drive access despite the fixed 30-day app cookie, and measured latency/revision-retention limitations. Do not mark performance targets passed or promise Drive revision retention. Unrestricted Console publication is a separate human-managed decision.
- Public health/HSTS, cookie attributes, sign-in/save/read/trash, restart refresh, navigation/sign-out and rollback/restoration have recorded evidence; public Disconnect/reconnect is human-reported. Visible application/HTTP callback logs omit queries and no log drain is enabled; undocumented internal platform retention is not proven. Consult the tracked release evidence before reopening gates.
- Explicit deferrals: keep Monday week start (configurability shelved), accept device-toolbar coverage instead of physical-phone testing for this stage, allow closed-tab backup expiry, and keep Appearance out of Settings.

## Working discipline and privacy

- Continue authorized reversible work autonomously, with concise updates. Prepare concrete code/tests before human Google steps. Do not invent results or silently replace locked tools/scopes/architecture.
- Follow the types in [Semantic Commit Messages](https://gist.github.com/joshbuchea/6f47e86d2510bce28f8e7f42ae84c716), with this project's subject format for commits and PR titles: `<type>: [<scope>] <subject>` when a scope is useful (for example, `feat: [google] integrate OAuth and Drive storage for localhost testing`), or `<type>: <subject>` without a scope. Use a concise present-tense subject; do not use parentheses around the scope. Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`. Apply the same classification to branch names as `<type>/<short-kebab-case-subject>` (a project convention extending the commit guide). PR descriptions explain the concrete change, validation, and material limitations; use semantic subjects when listing commits.
- Use GitHub **Squash and merge** when merging authorized PRs, unless the user explicitly requests another method. Use a semantic commit subject for the resulting squash commit.
- Inspect git status and preserve staged/user edits. Do not commit, push, publish, merge, or resolve remote threads without task authorization. Do not delegate unless explicitly requested by the user or applicable instructions.
- Keep README focused on durable product, setup and usage guidance; retain make generate in its quick start and document generator prerequisites. Keep evolving development/release status in PROGRESS.md/findings.md. Record work/evidence/remaining checks there; use ADRs for durable decisions. Distinguish requirements, automated results, human reports, waivers, and release conditions. A backlog is not authorization to implement it all.
- Never expose/commit `.env`, secrets, tokens, authorization codes, callback queries, whole cookies, account claims, or journal content. Logs contain method, query-free path, status, duration, and request IDs. Preserve query-free request logging before enabling real OAuth.
- Production content is not persisted server-side beyond bounded preview/index memory; local filesystem storage is a development exception. Preserve text through reconnect without silently switching accounts or navigating away before recovery is safe.

Calendar rendering accepts only the latest requested month, including failure feedback. Settings disconnect calls provider revocation before clearing the session, then reuses acknowledged sign-out draft cleanup; failures retain session/drafts and cleanup-only retries never repeat revocation. Local disconnect preserves saved files. Google Disconnect/reconnect has integrated live evidence and a human-reported public smoke pass; local-only results still do not establish Google revocation behavior.

## Go style and boundary conventions

- Write idiomatic Go that favors clarity, simplicity, and maintainability. Avoid magic strings/numbers, duplicated policy, hidden side effects, ambiguous arguments, and unnecessary abstractions. Reuse existing constants and standard-library facilities; name meaningful policy values near their owner without extracting every literal into a global constants package.
- Preserve clean architecture: dependencies point inward, domain rules remain independent of transport and infrastructure, adapters implement inward-facing ports, and the composition root wires concrete dependencies. Keep handlers thin. Introduce interfaces at genuine boundaries rather than for every concrete type.
- Load and validate configuration at startup, then pass explicit dependencies to constructors. Avoid hidden environment reads and constructors that guess argument meanings. Copy mutable inputs when retaining them would expose internal state to caller mutation.
- Validate domain invariants at the use-case boundary before storage or external calls. Keep serialization and parsing consistent, test boundary values and round trips, and preserve existing data when changing representations.
- Centralize shared policy while keeping presentation-specific behavior at the delivery edge. Avoid both duplicated validation and broad helpers that couple unrelated responsibilities.
- Handle errors explicitly, preserve error identity with wrapping, and use errors.Is/errors.As for inspection. Distinguish absence from failure, avoid partial success responses, and expose safe, actionable feedback. Never suppress errors merely to simplify a success path.
- Use consistent structured logging with configured levels and bounded diagnostic categories. Do not log secrets, personal content, or raw external errors that may contain sensitive data.
- Propagate context through blocking operations, check cancellation during long work, and make lock waits cancelable. Bound caches, coordination state, concurrency, and external I/O. Preserve invalidation guards and state clearly which concurrency guarantees stop at process boundaries.
- Avoid repeated external work within an operation. Prefer scoped reuse and batching before persistent caching; measure request counts or performance and account for staleness, invalidation, and memory bounds.
- Prefer guard clauses, focused functions, explicit constructors, consistent import groups, useful exported API documentation, and compile-time interface assertions where they clarify adapter contracts. Place these assertions immediately below the implementing struct declaration; group multiple assertions for the same struct together, including contracts whose methods live in other files. Avoid speculative flexibility, oversized functions, and unrelated stylistic churn.
- Share meaningful test fixtures across packages only when they represent the same setup. Keep test support independent of production code, use standard Go initialism casing, and leave exact contract expectations and boundary-case literals explicit so tests do not derive their expected result from the implementation.
- Add focused regression tests for observable contracts, edge cases, cancellation, and concurrency. Adopt lint rules incrementally, respect generated-code ownership, and verify architecture boundaries alongside formatting, lint, and tests.
