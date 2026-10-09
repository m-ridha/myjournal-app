# Agent guide for myjournal-app

## Start here

This is the self-contained project guide, reconciled with the repository and user decisions on 2026-10-09. It distills the product, engineering, and UI requirements needed to work without private planning documents. Requirements and historical passing checks are not evidence that every feature is implemented or accepted. Inspect affected code before changing behavior or reporting completion.

Read relevant sources in this order:

1. The user's current instructions and this file. Later user decisions override earlier plans.
2. `docs/development/PROGRESS.md` and `docs/development/findings.md` for current status, observations, and remaining checks.
3. `docs/development/adr/` for durable architecture decisions; `docs/development/phase-1-review.md` for Google experiment evidence and recorded fallbacks.
4. `docs/api/openapi.yml`, affected source/tests, and applicable tracked checklists such as `docs/development/editor-recovery-checks.md`.

Use the current paths under `docs/development/`, `docs/api/`, and `internal/web/api/`. Do not require private notes or historical chat exports for onboarding, or copy them wholesale into shared documentation. If a requirement is not settled here, in tracked docs, or by the conversation, identify the concrete gap rather than inventing it.

## Product and scope

`myjournal` is a quiet, responsive app for one plain-text journal entry per day, intended for friends and family. Production is a thin, stateless bridge to the user's own Google Drive. The local provider is development-only.

- Store UTF-8 plain text with LF line endings at `myjournal/entries/<year>/YYYY-MM-DD.txt` in a visible Drive folder. No JSON wrapper, metadata in the body, or rich-text rendering.
- Request only `openid`, `email`, `profile`, and `https://www.googleapis.com/auth/drive.file`. Do not broaden access to solve visibility problems.
- Exactly one entry per date. Past dates remain editable without a cutoff; future dates are rejected. Browser IANA time zone defines today. Once selected, the entry date stays fixed even if a save finishes after midnight.
- Delete moves the file to trash with confirmation. Reads never create missing folders; writes recreate missing paths. Identify Drive folders by private `appProperties`, not names alone. Renaming a folder must not break lookup.
- Keep provider connections in encrypted, authenticated HttpOnly cookies. No production user database or server-side session store. Temporary bounded caches and the local filesystem adapter do not change that goal.
- v1 is online-only. Draft recovery handles failed saves and expired sessions; it is not offline synchronization.
- English is the only v1 language. Keep all UI copy in the locale catalog; journal content may use any language.
- No additional providers, attachments, Markdown/rich text, client-side content encryption, search, location tags, sharing, reminders, AI summaries, native apps, or analytics collecting journal content. Park future ideas in `docs/development/ideas.md` without speculative scaffolding.

The planned v1 flow is: explain requested access, connect Google, write today or a past date, manually save and auto-save safely, browse a paginated newest-first list and month calendar, resolve conflicts, delete to trash, and manage sign-out/disconnect. Settings should show the connected account and a provider-supplied folder link; disconnect revokes access while leaving files intact. Privacy and Help are public, and Help must explain why files added manually may be invisible under `drive.file`.

## Working with the user

- Continue authorized, reversible phase work autonomously. Keep updates concise and ask only about genuinely unsettled requirements or dependencies.
- Prepare concrete code, tests, and run instructions before asking for interactive Google sign-in or Console changes. Never invent credentials or live results.
- Do not silently replace locked tools, architecture, OAuth scopes, or storage decisions. Resolve routine choices from this guide and the conversation.
- Inspect `git status` before editing and preserve user changes, including staged changes. Do not commit, push, publish, merge, or resolve remote review threads without task authorization. A historical small-commit workflow is not authorization.
- Record completed work and outstanding acceptance in PROGRESS.md/findings.md; use short ADRs for durable decisions. Keep human reports, automated results, and proposals distinguishable.
- Report exact commands actually run and results. Prefer `make fmt lint test` or `go test ./...`; do not claim fresh verification from historical results.
- Do not spawn agents unless the user or applicable instructions explicitly request delegation. When authorized, assign file ownership and avoid concurrent edits to schemas, generated code, and shared documentation.
- A backlog entry is not an instruction to implement the whole backlog. For a general continuation request, prioritize text safety and current phase acceptance.

## Repository map and boundaries

| Path | Responsibility |
| --- | --- |
| `cmd/app/` | Startup, configuration validation, provider registration, Wire composition root |
| `internal/journal/` | Domain types, `Store`/`Clock` ports, journal use cases, resolved time-zone context |
| `internal/journal/storetest/` | Shared storage contract tests |
| `internal/provider/provider.go` | Provider connection interface and registry |
| `internal/provider/local/` | Development sign-in and filesystem store |
| `internal/provider/google/` | Incomplete Google adapter, disabled in the app |
| `internal/cachestore/` | Shared bounded in-memory index/preview cache |
| `internal/clock/` | System clock adapter |
| `internal/session/` | AES-GCM cookie codec |
| `internal/config/` | Environment parsing and validation |
| `internal/web/`, `internal/web/handler/` | chi routes, pages, session checks, thin generated-interface API handlers |
| `internal/web/api/openapi.gen.go` | Committed generated API types and chi server |
| `internal/web/middleware/` | Focused middleware; currently request time-zone resolution |
| `internal/i18n/`, `assets/locales/en.json` | Embedded English locale catalog |
| `assets/templates/`, `assets/static/` | Embedded templates, CSS, JavaScript, favicons, SVG assets |
| `docs/api/openapi.yml` | HTTP contract source |
| `tests/` | Dependency-free browser interaction regressions |
| `spike/` | Standalone S01/S02/S03/S04/S07 experiments with README run guides |

Dependencies point inward. `internal/journal` must not import HTTP, chi, rendering, OAuth, Google APIs, filesystem APIs, or database packages. Adapters implement its ports and map errors to domain sentinels. `journal.Version` is opaque; the core/web must not interpret provider version fields. Keep Google tokens and resource details out of generic domain code.

`journal.Service` owns open/save/delete/list/calendar use cases. Handlers parse requests, obtain a session-scoped service, call it, and encode/render results. The injected service factory resolves provider/store/cache using the decoded connection. Production web/session code must not construct or import concrete providers, cache adapters, or system clocks. Wire them in `cmd/app`; providers must not import web. Keep `internal/web/handler` singular and generated API code inside `internal/`.

`internal/journal/store.go` holds the context-aware `Store` and `Clock` ports. Use the existing `journal.Clock` and `internal/clock.System`, not another utilities abstraction. Map wrapped errors with `errors.Is`. `.golangci.yml` expresses core/driving/driven depguard boundaries; deliberate-violation canaries remain unverified, so green lint alone does not prove all patterns match.

## Generation, configuration, and verification

- Preserve module `github/mridha/myjournal-app` and the declared Go version (currently 1.27.1). The unusual module spelling is not authorization to rename it.
- Keep Google Wire: edit `cmd/app/wire.go`, regenerate `cmd/app/wire_gen.go`, and version both. Never hand-edit generated composition code.
- Keep oapi-codegen: edit `docs/api/openapi.yml`, regenerate `internal/web/api/openapi.gen.go`, and inspect the generated diff. The schema declares OpenAPI 3.2.1; the generator header identifies v2.8.0. Verify installed compatibility instead of silently downgrading.
- API routes are `/api/v1/...`; `/healthz` is unversioned. Browser page and spike routes are separate contracts.
- Prefer the standard library and existing dependencies. No new runtime framework or unnecessary abstraction.
- `make run` loads `.env`; `go run ./cmd/app` does not. Refer to `.env.example` without printing `.env`.
- Defaults: port 3030, base URL `http://localhost:3030`, `PROVIDERS=local`, local data `./.devdata`, info logging, insecure cookies for HTTP localhost. `PORT` and `APP_BASE_URL` are separate and must stay aligned.
- `SESSION_KEY` is base64-encoded 32 random bytes, generated with `openssl rand -base64 32`. Startup rejects invalid keys, unknown/duplicate providers, and local mode outside exact localhost/loopback hostnames.
- `PROVIDERS=google` intentionally fails startup until real OAuth/Drive implementation and validation pass. Working spikes do not justify removing this guard.
- Spikes use port 3030 and `/auth/google/callback`; the app uses `/api/v1/auth/{provider}/...`. Resolve callback URL/config alignment when implementing OAuth. Do not kill an existing listener without establishing ownership.

```sh
make generate-openapi  # oapi-codegen on PATH
make generate-wire    # wire on PATH
make generate         # both
make run
make fmt lint test
make test-js          # only browser interaction regressions
```

`go generate ./cmd/app` also regenerates Wire. Regenerate only for relevant source changes. Lint uses golangci-lint v2. `make test` runs JavaScript regressions and `go test ./...`; JavaScript uses Node.js or macOS JavaScriptCore, without npm packages. The JavaScriptCore runner requires success markers because an asynchronous failure can otherwise yield an incorrect shell success.

Run `make fmt lint test` before calling implementation complete. For docs-only changes, verify paths, internal consistency, and `git diff --check`; new runtime tests are unnecessary. External-service tests must be mocked and repeatable. Use fixed clocks, temporary directories, and the shared `storetest` suite for every provider/cache adapter. Keep optional live Google runs separate from ordinary tests.

## UI, localization, and accessibility

- Use Go `html/template` and embedded assets with the existing Monochrome-inspired styling. Hugo is not an app build/runtime dependency. A verified theme port/source version/license record remains open; preserve applicable licenses when copying assets.
- Current interactions use a JSON API and vanilla JavaScript. HTMX/full-page-versus-fragment rendering was in the original plan; neither migration nor a formal deviation is complete. Do not claim HTMX is implemented or add a frontend framework silently. If adopting HTMX, pin/vendor it and handle fragments plus `Vary: HX-Request` together.
- Keep the editor initially empty, without sample content or placeholder text. Use plain text, readable system typography, preserved line breaks, and no horizontal overflow for long text/emoji.
- Header: brand, centered Write/Entries/Calendar links, shared light/dark toggle, signed-in Settings gear, and Privacy/Help. Preserve narrow-screen edge spacing and wrapping. Footer: centered `Built for quiet writing.`; do not duplicate Privacy/Help there.
- Appearance was removed from Settings by user decision. Do not restore System/Light/Dark controls there. The header toggle owns theme switching; review automatic device-theme following separately without reinstating that UI.
- Use the PNG favicon under `assets/static/favicons/`. Sun, moon, and gear SVGs live in `assets/static/svgs/`, not inline in HTML. CSS masks inherit `currentColor` for light/dark modes. The gear has six evenly spaced identical teeth; icon controls retain localized labels/tooltips.
- Use theme variables for text, surfaces, borders, and status colors. Keep visible keyboard focus, labelled controls, live-region status, disabled semantics, and adequate touch targets. Color must not be the only signal for saved/error state, calendar entries, or today.
- Aim for WCAG AA contrast, keyboard usability, 200% zoom, reduced-motion support, and long-entry wrapping. Proposed mock layouts are not automatically accepted behaviors; make focused changes to the existing design.
- Save feedback is noticeable and says `Saved`, not `Saved locally` or a mandatory timestamp. Editing after a save restores neutral grey state. Today's next-date control is visibly disabled; invalid/future page routes use the styled 404 layout.
- Previews start with the first meaningful line after leading whitespace. Preserve that behavior when fixing UTF-8-safe truncation. Lists are newest-first with year grouping; pagination should offer Newer/Older rather than an unverified total count.
- `assets/locales/en.json` is a flat map with dotted keys. Do not add nesting or load-time flattening. Templates use `t`; scripts use `data-messages`. New labels, statuses, errors, tooltips, and accessibility copy belong in that catalog. Dates, account labels, and journal text are runtime data. Missing translations currently display their keys.
- Escape user text through `html/template` or safe DOM text APIs; never mark journal content as trusted HTML. No third-party page requests, remote fonts, trackers, or CDN assets.

## Editor and date invariants

These are implemented and human-tested for the reported scenarios. Preserve them while adding other features:

- New backups use account/date-scoped `mj-draft:v3` keys in `sessionStorage`. Separate tabs have independent drafts; reload and same-tab navigation recover them. The user explicitly accepted that closing a tab can end recovery. Do not silently promise fresh-tab recovery or switch back to shared date-only backups.
- Draft scope is derived from provider/account identity without exposing credentials or account labels. Older v2 local backups and date-only backups require explicit Review older draft confirmation; retain known base versions and remove the old copy only after the new backup succeeds.
- Empty/whitespace edits are real drafts. Preserve text after failed loads/saves, HTTP 401, conflicts, and ambiguous acknowledgements. Storage exceptions must never claim a backup succeeded; advise keeping the page open or copying text.
- The initial HTML editor is read-only and Save disabled until GET returns usable text/version. Failed load offers Retry. Typing remains enabled during PUT; only acknowledged snapshots become clean, and later edits remain Unsaved changes with the returned version.
- Conflict actions survive reload of a stale draft. Keep mine fetches the latest version then conditionally saves. Load the other version confirms before replacing text and makes no PUT. Keep both appends saved text below local text and stays unsaved for review. Another writer can still cause another conflict.
- Sign-out probes storage access without clearing drafts, then requests logout. Only success triggers cleanup and the final cross-tab notification. Pending/failed logout preserves backups and editor usability. Cleanup failure after confirmed logout has a distinct signed-out message and cleanup-only retry. Preserve other accounts' backups and theme preference. Suspended tabs/storage failures can limit cross-tab cleanup; see ADR 011.
- `mj_tz` stores the browser's URL-encoded IANA location. Middleware resolves once per request; absent/invalid/oversized/host-dependent Local values fall back to UTC. `cmd/app` embeds `time/tzdata`. Pages, calendar defaults/markers, and future-date checks share that location and the injected clock.
- A synchronous bootstrap corrects the cookie before editing begins. Successful correction reloads the same date-dependent URL; blocked cookies must not create a loop. Default editor routes become `/entries/YYYY-MM-DD`; explicit dates and draft/save targets never shift at midnight. A loaded page's today marker is a snapshot until navigation/reload.
- Autosave is not implemented. The recorded fallback plan is five-second idle debounce, unchanged-content suppression, one save in flight, and serialization of follow-up saves against returned versions. Blur/page-hide network saves require careful acknowledgement, request-size, and draft-recovery handling; visibility changes currently back up drafts only.

## Current status and experiment evidence

Phase 0 and the restructuring are implemented. Much of Phase 2/3 works in local mode, but neither phase is fully accepted and the app is not production-ready. Read current findings for detailed evidence instead of repeating previously passed checks unnecessarily.

- Local login, protected pages/APIs, service selection by provider/account, manual writing, date navigation, list/calendar, recovery, conflict choices, sign-out, and header theme/Settings navigation are wired.
- Human reports accepted the editor recovery checklist, all eight conflict/tab-backup/Settings checks, failed-logout recovery, Jakarta midnight/date/calendar behavior, cookie correction, Safari usage, and developer-device-toolbar layouts. Browser/version details were not supplied for every report. Actual physical-phone testing was explicitly waived for this stage; do not reinstate it as a blocker without a new user decision.
- Local versions are SHA-256 content hashes; stale saves and save-after-delete are rejected. Atomic replacement, concurrent check/write safety, and collision-safe repeated trash remain open.
- Pagination accepts pages 1–10000 and sizes 1–100, validates before storage reads, and retains only page-size-plus-one dates. Previews are page-only and year reads stop when HasNext is known. This does not bound provider year-index allocations or eliminate deep-page scan work.
- Wire supplies one shared two-minute TTL/2048-item LRU cache keyed by provider/account. Preview size participates in the key; mutations invalidate account data; generation counters prevent stale fills; Get bypasses this cache. Redis is deferred.
- AES-GCM session round-trip/tamper behavior is tested. The fixed 30-day lifetime is adopted (ADR 002), but the app currently checks only that IssuedAt exists; server age enforcement and production cookie expiry remain pending. Refresh must not extend the original deadline. Shortening the lifetime later must also apply to existing cookies.
- The Google provider still returns placeholder auth data and has no Drive storage implementation. It remains disabled.

Google evidence and decisions are recorded in findings.md, phase-1-review.md, and ADRs 002/003/005:

- S1: test-account OAuth/ID-token validation, refresh, and Drive listing passed; subsequent consent-free sign-in omitted a refresh token. Preserve an existing token when a later response omits it.
- S2: tagged folder creation, rename-stable ID lookup, narrow-scope visibility, trash detection, safe recreation, and moved-folder discovery passed in the later reported run. Earlier failed trash/recreation runs are superseded evidence, not a reason to delete existing Drive folders automatically.
- S3: five naive workers created five paths; a shared in-process lock converged on one folder per role. This proves single-process serialization, not multi-instance convergence. Use a shared per-account lock and stable oldest-wins selection; preserve duplicate/data-bearing folders.
- S4: choose Drive `md5Checksum` as the content version (ADR 005): edits changed it, API/manual renames did not. A pre-upload metadata check is not atomic and misses A→B→A history; missing checksums need explicit handling. Production write safety remains unresolved.
- S7: tested encrypted-cookie refresh/cache lifecycle, cold restart, tamper/key rotation, sign-out, token preservation, and revoked-access invalid_grant behavior passed. Measured cookie header: 584 bytes, comfortably below 3800. Chrome 154.0.8037.98 arm64 accepted Secure on HTTP localhost; Safari 26.4 required COOKIE_SECURE=false there. Production HTTPS acceptance remains separate.
- S5/S6: revision and latency experiments are deferred with documented conservative fallbacks, not measured successes. Google previews should cap concurrency at four, bound reads, consume listing pagination, and use bounded cancellation-aware backoff/jitter. Do not blindly retry ambiguous folder/file creation; requery tagged paths first.
- S11/S12: production OAuth/domain/release dependencies and library choices are recorded in phase-1-review.md. Testing-token expiry cannot be extended by a 30-day browser cookie. Verify current Console/official requirements before release; no production publishing or warning-free consent is established.

The user gave a Phase 1 go decision on 2026-10-08. Phase 4 still depends on completing Phase 2/3 gates and validating actual Google implementation; do not rerun accepted S1/S2/S4/S7 experiments as routine prerequisites.

## Phase acceptance and remaining priorities

Use this checklist with PROGRESS.md/findings.md. Historical plans do not override the explicit deferrals: configurable Monday/Sunday week start is shelved as nice to have (keep Monday), physical-phone checks are waived for this stage, closed-tab draft recovery is not required, and Settings Appearance controls were removed.

1. **Phase 2 — core/local correctness:** pure entry paths/date parsing and leap-day tests; LF normalization and valid UTF-8 handling; user-zone dates with injected midnight/DST tests; future-date rejection; stale-save/create/delete conflicts; newest-first cross-year pagination; calendar markers; provider registry/config boundaries; bounded concurrent cache behavior; shared store contracts for local and cache-wrapped stores. Add atomic local replacement and protection across stores targeting the same files, plus safe repeated trash and UTF-8 previews. Verify depguard deliberately and remove the canaries afterward.
2. **Phase 3 — complete local flow:** retain the accepted editor/date/conflict/sign-out behaviors; add safe autosave/blur/page-hide behavior, delete confirmation, list pagination controls, disconnect/revoke explanation, and folder-link/settings/help details. Avoid out-of-order calendar responses. Align the API schema/handlers/client, error bodies, baseVersion and missing-entry semantics; map wrapped reconnect/unavailable errors. Add session-age validation, cross-origin/CSRF protection, bounded request bodies, path-only logging, and focused accessibility checks. Record S8 date/session, S9 theme provenance, and S10 save safety evidence, including actual deviations rather than claiming unimplemented HTMX fragments or a verified theme port.
3. **Phase 4 — Google adapter:** implement state/PKCE authorization-code flow and validated ID claims, preserve refresh tokens, keep credentials versioned/small, and cache access tokens only in bounded process memory. Use tagged folder lookup/recreation and duplicate-preserving selection, context-aware Drive storage, domain error mapping/backoff, and token revocation. Align real callback routes. Pass shared contracts against mocked Drive and human-assisted write/list/calendar/edit/trash/rename/revocation checks before accepting Google mode. Avoid leaking adapter details into core/web.
4. **Phase 5 — reliability:** validate all conflict choices across clients against Google, created/deleted-elsewhere cases, outages/revocation, external Drive changes, cache expiry/invalidation, and safe autosave. Measure performance instead of inferring it from a single spike sample. Targets are today's load <1.5 s, manual save <1 s, cached calendar navigation <500 ms at p75; record gaps and plans if unmet.
5. **Phase 6 — release:** audit logging/privacy/security, run govulncheck, apply HTTPS/Secure/HSTS and a CSP compatible with the remaining inline theme script, and document a reproducible non-root deployment. Human handles hosting/domain/Console publishing. Confirm consent/homepage/privacy URLs, release callback alignment, applicable verification, and actual production sign-in/refresh before inviting users. No journal-content analytics.

The next agreed priority is Phase 2 storage correctness. Other rechecked gaps include shutdown that does not wait for drain completion; HTTP server timeouts beyond ReadHeaderTimeout; session rotation/age/CSRF; and automatic device-theme following (theme.js currently persists the initial device preference as an explicit choice). Removing Appearance did not establish automatic device-theme behavior.

Use the adopted standard-library net/http.CrossOriginProtection direction for unsafe browser requests, with meaningful failure tests; OAuth callbacks still need explicit state/PKCE validation. Do not treat SameSite alone as CSRF coverage. Security headers must be compatible with actual assets and scripts. The simplified provider/auth seam and callback handling still need implementation work, not broader scopes or a new framework.

## Privacy and verification discipline

Never commit or expose `.env`, client secrets, session keys, refresh/access tokens, authorization codes, complete callback URLs, whole cookies, or journal content. Shared docs should contain observable flags/counts, not account claims or sensitive chat snippets. Logs should include method, path without sensitive queries, status, duration, and request identifiers. Current chi logging can include OAuth query parameters; replace it before enabling real OAuth.

Production journal content is not persisted on the server; request memory and bounded preview/index caches are allowed, and the local filesystem provider is an explicit development exception. Browser backups must match their documented tab lifecycle. Preserve user text on auth/storage failures and provide an explicit reconnect path without silently switching accounts or navigating away before recovery is safe.

Mock external services for normal tests. Live Google runs remain human-assisted and use synthetic files/content. Distinguish repeatable tests, reported browser checks, waivers, and unverified release conditions. Report material remaining acceptance checks without presenting historical passing results as a new run.
