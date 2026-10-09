# Agent guide for myjournal-app

## Start here

This guide contains durable instructions and user decisions. All required context is in this repository; private planning files and historical chat exports are not prerequisites. Inspect affected code before treating a requirement or historical passing check as current behavior.

Read the user's current instructions first, then consult:

- [Progress](docs/development/PROGRESS.md) and [findings](docs/development/findings.md) for status and evidence.
- [Phase acceptance](docs/development/phase-acceptance.md) for detailed requirements and remaining gates.
- [ADRs](docs/development/adr/) for architecture decisions and [Phase 1 review](docs/development/phase-1-review.md) for Google evidence/fallbacks.
- [OpenAPI](docs/api/openapi.yml), affected code/tests, and the [recovery checklist](docs/development/editor-recovery-checks.md) as relevant.

## Stack, setup, and checks

Use Go, chi, `html/template`, embedded assets, Google Wire, and oapi-codegen. Preserve module `github/mridha/myjournal-app` and the declared Go version (currently 1.27.1). Prefer the standard library and existing dependencies.

```sh
# Create .env from .env.example; set SESSION_KEY using openssl rand -base64 32.
make run               # loads .env; go run ./cmd/app does not
make fmt lint test     # required for implementation changes
make test-js           # only browser interaction regressions
make generate-openapi  # after OpenAPI edits; requires oapi-codegen
make generate-wire     # after Wire edits; requires wire
```

- Defaults: port 3030, `APP_BASE_URL=http://localhost:3030`, `PROVIDERS=local`, `LOCAL_DATA_DIR=./.devdata`, info logging, insecure cookies on HTTP localhost. Keep listener port and base URL aligned; they are separate settings.
- `SESSION_KEY` is base64-encoded 32 random bytes. Startup rejects invalid keys, unknown/duplicate providers, and local mode outside exact localhost/loopback hostnames. Never print `.env`.
- Edit `cmd/app/wire.go`, regenerate `cmd/app/wire_gen.go`, and version both. Edit `docs/api/openapi.yml`, regenerate `internal/web/api/openapi.gen.go`. Never hand-edit generated code or restore top-level generated directories.
- The schema declares OpenAPI 3.2.1 and the generated header identifies oapi-codegen v2.8.0. Verify installed compatibility; do not silently downgrade. `make generate` runs both generators; `go generate ./cmd/app` also runs Wire. Inspect generated diffs.
- `make test` runs JavaScript regressions and `go test ./...`. JavaScript requires Node.js or macOS JavaScriptCore, without npm packages. Preserve the JavaScriptCore success-marker check so asynchronous assertions cannot appear to pass.
- Lint uses golangci-lint v2. For documentation-only changes, check links/paths, consistency, and `git diff --check`; runtime tests are unnecessary. Report unavailable tools and exact commands/results; never present historical checks as a fresh run.
- Spikes also use port 3030. Do not kill an existing listener without establishing ownership.

## Product constraints

`myjournal` is a quiet app for one plain-text entry per day. Production is a stateless bridge to the user's Google Drive; local storage is development-only.

- Files are UTF-8 plain text with LF endings at `myjournal/entries/<year>/YYYY-MM-DD.txt`. No JSON wrapper, body metadata, or rich-text rendering.
- Request only `openid`, `email`, `profile`, and `https://www.googleapis.com/auth/drive.file`. Never broaden scopes to fix visibility.
- Past dates remain editable without a cutoff; future dates are rejected. Browser IANA time zone defines today; a selected date never shifts at midnight.
- Delete means trash, with confirmation. Reads never create folders; writes recreate missing paths. Discover folders by private `appProperties`, not names. Preserve renamed and duplicate/data-bearing folders.
- Connections belong in encrypted, authenticated HttpOnly cookies. No production user database or server-side session store; bounded in-memory caches are allowed. Keep Redis deferred.
- v1 is online-only and English-only. Browser backup is recovery, not offline sync. No additional providers, attachments, Markdown, client-side encryption, search, tags, sharing, reminders, AI summaries, native apps, or journal-content analytics. Park ideas in `docs/development/ideas.md`.

## Architecture

| Path | Responsibility |
| --- | --- |
| `cmd/app/` | Configuration, provider registration, Wire composition root |
| `internal/journal/`, `internal/journal/storetest/` | Domain/use cases, Store/Clock ports, shared contracts |
| `internal/provider/`, `internal/provider/{local,google}/` | Connection port/registry and storage adapters |
| `internal/cachestore/`, `internal/clock/` | Bounded cache and system clock adapters |
| `internal/session/`, `internal/config/` | AES-GCM cookie codec and environment validation |
| `internal/web/`, `internal/web/handler/`, `internal/web/middleware/` | Router/pages, thin API handlers, focused middleware |
| `internal/web/api/openapi.gen.go` | Committed generated API code |
| `internal/i18n/`, `assets/` | Locale catalog, embedded templates/static files |
| `tests/`, `spike/` | Browser regressions and standalone Google experiments |

- Dependencies point inward. Domain code must not import HTTP/chi, rendering, OAuth, Google APIs, filesystem, or database packages. Providers must not import web.
- `journal.Service` owns open/save/delete/list/calendar. Handlers parse requests and call the injected session-scoped service factory. Only `cmd/app` wires concrete providers, cache adapters, and system clocks into production web/session code.
- Keep Store/Clock in `internal/journal/store.go`; use `journal.Clock` and `internal/clock.System`, not a new utilities abstraction. Store methods are context-aware and `journal.Version` is opaque. Map wrapped errors with `errors.Is`; do not expose provider details to generic code.
- Preserve shared account-isolated caches, invalidation/generation guards, bounded memory, page-only previews, and Get bypass. Pagination validates pages 1–10000/sizes 1–100 before storage access and retains at most size+1 dates.
- Run shared store contracts for every adapter. Use fixed clocks, temporary directories, and mocked external services. Depguard's deliberate-violation canaries remain unverified; green lint alone does not prove all boundary patterns match.
- APIs use `/api/v1/...`; `/healthz` is unversioned. Browser routes are separate. Current interactions use JSON and vanilla JavaScript; the original HTMX plan still needs migration or a recorded deviation. Do not claim fragments are implemented or silently introduce a framework.

## UI and localization

- Preserve Monochrome-inspired responsive styling and licenses. Hugo is not a runtime/build dependency; verified theme-port provenance remains an acceptance task.
- Keep the initial editor empty, without sample content or placeholder. Use readable typography, plain text, preserved line breaks, and wrapping for long text/emoji.
- Header: brand, centered Write/Entries/Calendar, theme toggle, signed-in Settings gear, Privacy/Help. Keep narrow-screen spacing/wrapping. Footer is centered `Built for quiet writing.` without duplicate links.
- Appearance was removed from Settings by user decision. Do not restore System/Light/Dark controls there; the shared header owns switching. Automatic device-theme following is a separate open issue.
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
- Stale recovered drafts retain conflict actions. Keep mine fetches latest then conditionally saves; Load the other version confirms replacement and makes no PUT; Keep both appends saved text below local text for review before Save. Further changes can conflict again.
- Logout must succeed before draft deletion/final cross-tab notification. Pending/failed logout preserves recovery and editor usability. Post-logout cleanup failure gets a truthful signed-out message and cleanup-only retry. Preserve other accounts/theme; see ADR 011 for notification/storage limits.
- Resolve URL-encoded IANA `mj_tz` once per request, defaulting missing/invalid/oversized/Local to UTC; keep embedded tzdata. Pages, calendar, and future checks share the resolved location and injected clock.
- Correct stale timezone cookies before editing, without reload loops when storage is blocked. Canonicalize default editor routes to an explicit date; reload preserves it. A loaded page's today marker is a snapshot until navigation/reload.
- Autosave is pending: the recorded plan is five-second idle debounce, unchanged-content suppression, one request in flight, and serialized follow-up saves against returned versions. Blur/page-hide saves must preserve these invariants; visibility changes currently back up drafts only.

## Google and current priorities

- Production Google is incomplete and intentionally disabled. Do not remove the startup guard based on spike results. S1/S2/S4/S7 and the Phase 1 go decision are recorded; consult tracked evidence instead of routinely asking to repeat accepted experiments.
- Use Drive md5Checksum for content versions (ADR 005), while treating pre-upload comparison as non-atomic and handling absent checksums explicitly. Share folder locks per account; select duplicates deterministically without deleting them. Single-process convergence does not prove multi-instance safety.
- Preserve existing refresh tokens when later OAuth responses omit them. The approved lifetime is fixed 30 days from original IssuedAt (ADR 002); refresh must not extend it. Server age and cookie expiry are enforced; a shorter future limit must apply to existing cookies too.
- Spikes use `/auth/google/callback`; app auth uses `/api/v1/auth/{provider}/...`. Align real callback/configuration and implement state, PKCE, code/claim validation before enabling Google. Keep live checks human-assisted and production Console/HTTPS acceptance separate.
- Local atomic saves, same-process mutation protection across stores, repeated trash preservation, and UTF-8 validation/previews are implemented. Next priorities: remaining security/lifecycle and API acceptance. Remaining security/API/UI/release gates are in the tracked phase checklist; Phase 2/3 are not fully accepted.
- Use the adopted net/http.CrossOriginProtection direction for unsafe browser requests; SameSite alone is insufficient, and OAuth still needs state/PKCE. Session age, a 1 MiB save-envelope cap, and path-only logging are implemented. Address timeouts/draining and CSP-compatible scripts in focused tasks.
- Explicit deferrals: keep Monday week start (configurability shelved), accept device-toolbar coverage instead of physical-phone testing for this stage, allow closed-tab backup expiry, and keep Appearance out of Settings.

## Working discipline and privacy

- Continue authorized reversible work autonomously, with concise updates. Prepare concrete code/tests before human Google steps. Do not invent results or silently replace locked tools/scopes/architecture.
- Follow [Semantic Commit Messages](https://gist.github.com/joshbuchea/6f47e86d2510bce28f8e7f42ae84c716): commit subjects and PR titles use `<type>(<optional scope>): <subject>` with a concise present-tense subject. Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`. Apply the same classification to branch names as `<type>/<short-kebab-case-subject>` (a project convention extending the commit guide). PR descriptions explain the concrete change, validation, and material limitations; use semantic subjects when listing commits.
- Inspect git status and preserve staged/user edits. Do not commit, push, publish, merge, or resolve remote threads without task authorization. Do not delegate unless explicitly requested by the user or applicable instructions.
- Record work/evidence/remaining checks in PROGRESS.md/findings.md; use ADRs for durable decisions. Distinguish requirements, automated results, human reports, waivers, and release conditions. A backlog is not authorization to implement it all.
- Never expose/commit `.env`, secrets, tokens, authorization codes, callback queries, whole cookies, account claims, or journal content. Logs contain method, query-free path, status, duration, and request IDs. Current chi query logging must be replaced before real OAuth.
- Production content is not persisted server-side beyond bounded preview/index memory; local filesystem storage is a development exception. Preserve text through reconnect without silently switching accounts or navigating away before recovery is safe.
