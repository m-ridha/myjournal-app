# Phase acceptance requirements

## Current status (2026-10-12)

This summary supersedes pending language in the dated evidence below; those records describe what was known at the time, not new work to repeat.

- Google OAuth/Drive registration and storage are implemented. Non-test registration is allowed with a valid HTTPS origin and Secure cookies; GOOGLE_TEST_MODE remains restricted to exact loopback/localhost URLs. Restricted Google Testing acceptance and the first v0.1.0 published-release deployment passed on 2026-10-10 (ADR 016). This is not unrestricted Console publication or branding verification.
- Theme initialization is already externalized to assets/static/js/theme-init.js, loaded before CSS. Same-origin CSP/security headers and host-only HTTPS HSTS are implemented (ADR 012). There is no remaining inline-theme migration task.
- Focused Chrome UI acceptance is complete with separately recorded human announcement evidence and device-testing waiver; coverage limits remain in [UI acceptance checks](ui-acceptance-checks.md). Full WCAG certification and blanket Phase 2/3 completion are not asserted.
- Google branding/ownership submission remains separate; see [branding verification](google-branding-verification.md). Accepted latency/revision-retention limits and non-atomic external-writer/deployment-overlap risks remain documented rather than treated as passed targets. No new Google experiments or deployment are implied by this reconciliation.

This is the repository-local acceptance checklist distilled from the project plan. It describes requirements, not a claim of completion. Consult [PROGRESS.md](PROGRESS.md), [findings.md](findings.md), [ADRs](adr/), and the [Phase 1 review](phase-1-review.md) for implemented behavior, reported evidence, and recorded fallbacks. Recheck code before reopening or closing a gate.

Later user decisions supersede earlier plans:

- English and Bahasa Indonesia are supported. Language preferences and Settings are public; account details and actions require a valid session. The shared header owns language and theme controls, with Privacy linked from Help.
- Footer/version/logging changes are implemented with automated evidence; final footer visual and deployed-build checks remain open. See the current summary in PROGRESS.md.

- Keep Monday first. Configurable Monday/Sunday week start is shelved as nice to have.
- Physical-phone testing is waived for this stage; developer-device-toolbar coverage is accepted. Do not describe it as physical-device evidence.
- Backups need reload/same-tab recovery; fresh-tab recovery after closing a tab is not required.
- Settings Appearance is removed; the header light/dark toggle remains.
- Keep the current module, Wire, and oapi-codegen. Retain JSON API and vanilla JavaScript; HTMX is excluded by user decision (ADR 014).

## Phase 0 and 1: bootstrap and experiments

Bootstrap/configuration and the recorded Phase 1 go decision support continuing local development. S1/S2/S4/S7 evidence and S3 single-process concurrency observations are recorded; do not equate them with a complete Google provider or repeat them routinely. S5/S6 use documented conservative fallbacks, while production latency, revisions, Console, and HTTPS checks remain separate. The initial registration guard was subsequently replaced with HTTPS-origin/Secure-cookie validation; restricted Google Testing release acceptance is recorded in the current-status summary above.

## Phase 2: core and local correctness

Goal: business rules work behind ports, with a reliable development provider.

- Pure entry paths/date parsing, year consistency, and edge cases such as leap days.
- LF normalization, valid UTF-8 input handling, and UTF-8-safe first-meaningful-line previews.
- Browser-zone dates, injected-clock midnight/DST coverage, future-date rejection, and fixed selected dates.
- Stale save, save-after-delete, and created/deleted-elsewhere conflict semantics.
- Newest-first lazy pagination across years; safe bounds; page-only previews and one-item lookahead.
- Calendar grids and entry/today/future markers using the same date authority, with Monday first.
- Provider registry/config validation and development-only local registration.
- Shared bounded account-isolated cache, concurrent invalidation correctness, precise TTL expiry and bounded BigCache eviction (ADR 017; replaces the custom 2048-entry LRU), and Get bypass.
- Shared store contracts for local and cache-wrapped storage; later Google storage uses the same suite.
- Atomic local replacement, concurrency protection across stores targeting the same files, and collision-safe repeated trash.
- Inward dependencies verified with temporary depguard violations for core/driving/driven rules; remove the canaries afterward.

Local atomic replacement, same-process mutation protection across store instances, and collision-safe repeated trash are implemented and regression-tested (2026-10-09). The lock serializes all local mutations; separate processes and external filesystem writers are not coordinated. UTF-8 validation, byte-safe meaningful-line previews, and local/cache-wrapped store contracts are also implemented. Accepted date/recovery behavior must remain intact. Depguard matching is now verified by thirty-one temporary forbidden-import probes via make test-architecture, including nested packages and concrete clock/cache wiring. Test-only adapter composition is explicitly exempt from the driving-side rule (2026-10-09).

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
- Document theme source/version/license provenance (ADR 004 now records the Monochrome visual-reference comparison; no copied theme source detected). The browser architecture decision is complete: JSON API and vanilla JavaScript (ADR 014); HTMX fragments and Vary: HX-Request are not requirements.
- Verify keyboard interaction, accessible labels/live status, focus, contrast in both themes, reduced motion, 200% zoom, long/emoji/unbroken text, and template execution. Respect the device-testing waiver while recording the exact coverage performed.
- Record S8 date/session, S9 theme, and S10 save-safety evidence and actual remaining limitations.

Fixed 30-day server-side age and cookie expiry, key-rotation rejection, unsafe-request cross-origin protection, a 1 MiB save-envelope cap, and query-free logging are implemented with automated checks (2026-10-09). HTTP timeouts/draining and same-origin CSP/security headers are implemented (ADR 012); Covered CSP flows passed in Chrome 155/native Safari 26.4 (see csp-browser-qa.md); Safari full startup-console history remains unverified. API error/input/cookie contract alignment is implemented with regression coverage (2026-10-09). Safe local autosave is implemented (ADR 013); its browser coverage and navigation limits are recorded in findings. Confirmed trash and paginated list controls are implemented with local regressions (delete-browsing-checks.md). Stale calendar-response protection and confirmed Settings disconnect are implemented with local browser and automated failure coverage (calendar-disconnect-checks.md). Integrated Google OAuth, the Settings folder link, and user-revoked-access draft recovery now have partial live coverage (2026-10-10); Settings Disconnect/reconnect and disposable-year recreation now have live coverage (2026-10-10); the remaining Google checklist stays open. Device-theme following until explicit selection and reduced-motion handling are implemented. Focused Chrome keyboard, Unicode wrapping, labeled controls, narrow-layout, and CSS 200% reflow checks passed; An additional 18 axe scans and keyboard reachability/indicator checks passed in both themes, including failed-save/conflict states, after dark Disconnect contrast and calendar naming fixes (2026-10-10). The focused native-zoom, keyboard/focus, non-text contrast and icon-provenance checklist is complete on 2026-10-11, with separately recorded human screen-reader announcement evidence and limitations (ui-acceptance-checks.md). This is scoped Chrome acceptance, not full accessibility certification or full Phase 2/3 completion. JSON API and vanilla JavaScript are adopted in ADR 014; htmx-migration-assessment.md preserves rejected migration research. Accepted recovery/conflict/date/logout scenarios do not complete the whole phase.

## Phase 4: Google adapter

Goal: validated Google sign-in and Drive-backed storage behind the existing boundaries.

Integrated implementation (2026-10-10): encrypted browser transaction, code exchange with PKCE, Google ID-token signature/claims/nonce checks, required Drive grant, same-account refresh preservation, bounded access-token caching, revocation, and all Drive Store methods now have synthetic automated coverage. Shared Store contracts pass against mocked Drive; per-request stores share bounded mutation lock stripes. Settings links an existing tagged root without creating one on reads. GOOGLE_TEST_MODE=true permits localhost integration only; production remains gated. User confirmed Console callback/API/test-user configuration; partial live Chrome acceptance covers sign-in, restart/read, synthetic Unicode storage, list/calendar, all conflict controls, external Drive revision conflicts, and user-revoked-access reconnect/save recovery. Synthetic entries were trashed afterward. The first reconnect callback rejection remains unexplained; remaining app acceptance is tracked in [google-live-checks.md](google-live-checks.md) and findings.

Checksum comparison and upload remain non-atomic against external writers. Optional ETag/If-Match forwarding is unverified against real Drive and does not close this gate. Single-process serialization is not multi-instance safety. ADR 016 records the user-approved restricted first-release policy (one active process, no simultaneous external edits, documented lost-update risk); deployment compliance and other release gates remain required. Later measured latency/revision limits and the restricted first-release acceptance are recorded above and in ADR 016. Original performance targets and revision-retention guarantees are not marked passed.

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

Synthetic reliability coverage (2026-10-10) now verifies bounded transient-read retries, cancellation/deadline during backoff, and applied-but-unacknowledged creates/updates without automatic mutation retry. Fresh reads recover applied content/version and stale original-base retries conflict. This is mocked evidence. Later live reliability measurements and the accepted restricted-release external-writer policy are recorded in ADR 016 and the dated findings; do not equate accepted limitations with passed performance targets.

## Phase 6: release

- Audit privacy/logging/error/panic paths; no secrets, full cookies, authorization queries, or journal content in logs. Run govulncheck and review dependencies.
- HTTPS/Secure cookies, production HSTS, verified cross-origin/CSRF protection, and a CSP compatible with the actual scripts/assets. Theme bootstrap is externalized in theme-init.js and covered by the adopted same-origin CSP (ADR 012).
- Document reproducible non-root single-binary deployment, environment configuration, local/test commands, architecture, and theme provenance.
- Human handles hosting/domain and Google Console publishing. Confirm callback alignment, homepage/privacy URLs, domain ownership/applicable verification, and real production sign-in/refresh. Testing token limits and consent warnings cannot be solved by increasing browser session lifetime.
- Verify accessibility/themes and the accepted device coverage; record release gaps. A fresh deployment from README must work, and the human confirms the release checklist.
- Observe onboarding and reliability without journal-content analytics; prefer direct feedback. Do not promise production readiness from completed local phases.
