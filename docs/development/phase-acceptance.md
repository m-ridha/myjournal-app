# Phase acceptance requirements

This is the repository-local acceptance checklist distilled from the project plan. It describes requirements, not a claim of completion. Consult [PROGRESS.md](PROGRESS.md), [findings.md](findings.md), [ADRs](adr/), and the [Phase 1 review](phase-1-review.md) for implemented behavior, reported evidence, and recorded fallbacks. Recheck code before reopening or closing a gate.

Later user decisions supersede earlier plans:

- Keep Monday first. Configurable Monday/Sunday week start is shelved as nice to have.
- Physical-phone testing is waived for this stage; developer-device-toolbar coverage is accepted. Do not describe it as physical-device evidence.
- Backups need reload/same-tab recovery; fresh-tab recovery after closing a tab is not required.
- Settings Appearance is removed; the header light/dark toggle remains.
- Keep the current module, Wire, and oapi-codegen. Do not silently settle the outstanding HTMX implementation decision.

## Phase 0 and 1: bootstrap and experiments

Bootstrap/configuration and the recorded Phase 1 go decision support continuing local development. S1/S2/S4/S7 evidence and S3 single-process concurrency observations are recorded; do not equate them with a complete Google provider or repeat them routinely. S5/S6 use documented conservative fallbacks, while production latency, revisions, Console, and HTTPS checks remain separate. Production Google registration stays disabled until implementation and validation pass.

## Phase 2: core and local correctness

Goal: business rules work behind ports, with a reliable development provider.

- Pure entry paths/date parsing, year consistency, and edge cases such as leap days.
- LF normalization, valid UTF-8 input handling, and UTF-8-safe first-meaningful-line previews.
- Browser-zone dates, injected-clock midnight/DST coverage, future-date rejection, and fixed selected dates.
- Stale save, save-after-delete, and created/deleted-elsewhere conflict semantics.
- Newest-first lazy pagination across years; safe bounds; page-only previews and one-item lookahead.
- Calendar grids and entry/today/future markers using the same date authority, with Monday first.
- Provider registry/config validation and development-only local registration.
- Shared bounded account-isolated cache, concurrent invalidation correctness, TTL/LRU behavior, and Get bypass.
- Shared store contracts for local and cache-wrapped storage; later Google storage uses the same suite.
- Atomic local replacement, concurrency protection across stores targeting the same files, and collision-safe repeated trash.
- Inward dependencies verified with temporary depguard violations for core/driving/driven rules; remove the canaries afterward.

The immediate priority is storage correctness. Accepted date/recovery behavior must remain intact. Passing sequential stale-save tests alone does not establish concurrent check/write safety.

## Phase 3: complete local flow

Goal: a usable and accessible local journaling flow, without requiring Google.

- Explain access on landing; local login, protected pages/APIs, friendly auth errors, and provider/account-scoped services.
- Manual save plus safe autosave: five-second idle debounce, unchanged-content suppression, one save in flight, follow-up saves against returned versions, blur/page-hide handling, and accurate acknowledgement/draft state.
- Preserve accepted draft recovery, tab isolation, initial-load lock, conflict actions, and failed-logout ordering. Use the [recovery checklist](editor-recovery-checks.md); new behavior needs new acceptance, not historical pass claims.
- Browse a paginated newest-first list with Newer/Older controls and a calendar with previous/next month, entry markers, and jump to today. Reject stale calendar responses during rapid navigation.
- Delete with confirmation; disconnect/revoke with an explanation that files remain; Settings account and provider-supplied folder link; public Privacy/Help including drive.file manual-file visibility limits.
- Enforce the adopted fixed 30-day session age, cookie lifecycle/rotation, and cross-origin/CSRF protection. Add bounded request bodies and appropriate timeouts/draining; log paths without OAuth queries.
- Align OpenAPI, handlers, and browser behavior: baseVersion, missing entries, validation statuses, response/error bodies, cookie security/401 documentation, and wrapped reconnect/unavailable errors.
- Retain the shared header theme toggle and gear link. Review initial device-theme following without restoring Appearance controls.
- Document theme source/version/license provenance and the actual HTMX-versus-JSON/vanilla-JavaScript decision. If HTMX is adopted, vendor a pinned version and implement full pages/fragments with Vary: HX-Request together. Do not claim these are already complete.
- Verify keyboard interaction, accessible labels/live status, focus, contrast in both themes, reduced motion, 200% zoom, long/emoji/unbroken text, and template execution. Respect the device-testing waiver while recording the exact coverage performed.
- Record S8 date/session, S9 theme, and S10 save-safety evidence and actual remaining limitations.

The accepted recovery/conflict/date/logout scenarios do not by themselves complete security, autosave, deletion, browsing controls, or the whole phase.

## Phase 4: Google adapter

Goal: validated Google sign-in and Drive-backed storage behind the existing boundaries.

- State/PKCE authorization-code flow, callback/config alignment, code exchange, friendly rejection/partial-grant handling, and validated ID-token signature/claims.
- Preserve refresh tokens omitted by subsequent responses; versioned small encrypted credentials and bounded in-memory access-token caching. No server-side token persistence.
- Onboard root/entries folders and supply a folder link; tagged lookup/recreation, shared per-account locks, oldest-wins duplicate selection, and preservation of other/data-bearing folders. Reads never create paths.
- Implement all Store methods with context cancellation, Drive-list pagination, narrow scope, domain error mapping, bounded backoff/jitter, and token revocation. Preserve checksum version opacity and explicitly address the non-atomic check/update race.
- Pass shared contracts against mocked Drive; test credentials, error mapping, backoff, duplicate paths, and unavailable/revoked access.
- Human-assisted test account: exact plain-text write at the expected path, list/calendar/past edit, trash, rename-stable folder discovery, revoked-access reconnect without losing typed text, and query-free/content-free logs.
- Enable Google only after implementation and validation pass. Local UI/spike success is not Google acceptance.

## Phase 5: reliability

- All three conflict choices across clients against Google, including created/deleted-elsewhere cases and another writer during resolution.
- Rate limits, outages, and mid-save failures retain text and provide retry/reconnect without incorrect saved state.
- Cache invalidation/expiry/manual refresh with external Drive changes; missing/trashed/renamed folders/files and list/get races.
- Tune autosave with actual revision measurements before making retention guarantees; Drive revisions are not draft recovery.
- Measure performance; targets at p75 are today's load under 1.5 seconds, manual save under 1 second, and cached calendar navigation under 500 ms. Record gaps and plans rather than treating one spike latency sample as acceptance.

## Phase 6: release

- Audit privacy/logging/error/panic paths; no secrets, full cookies, authorization queries, or journal content in logs. Run govulncheck and review dependencies.
- HTTPS/Secure cookies, production HSTS, verified cross-origin/CSRF protection, and a CSP compatible with the actual scripts/assets. Remaining inline theme setup needs a compatible policy or refactor.
- Document reproducible non-root single-binary deployment, environment configuration, local/test commands, architecture, and theme provenance.
- Human handles hosting/domain and Google Console publishing. Confirm callback alignment, homepage/privacy URLs, domain ownership/applicable verification, and real production sign-in/refresh. Testing token limits and consent warnings cannot be solved by increasing browser session lifetime.
- Verify accessibility/themes and the accepted device coverage; record release gaps. A fresh deployment from README must work, and the human confirms the release checklist.
- Observe onboarding and reliability without journal-content analytics; prefer direct feedback. Do not promise production readiness from completed local phases.
