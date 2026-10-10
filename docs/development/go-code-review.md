# Go code and architecture review — 2026-10-10

Review of the current project at cb43190, excluding private agents/ files. No runtime changes were made. Production code was inspected across cmd/app, journal, provider, cache, session, web, config, i18n, and embedded assets; test coverage and generated boundaries were inspected. Spike entry points were sampled separately as experimental code. This is a prioritized review, not a security audit or proof that every possible defect has been found.

## Verification

Fresh golangci-lint run (0 issues), go vet ./..., and go test -race -short ./... passed. All seven make test-architecture probes passed separately and were cleaned up. Temporary synthetic probes reproduced the date/path issues below and were removed. No live Google calls or journal changes were needed.

## Correctness and lifecycle fixes

1. **P2 — Cache generation bookkeeping is unbounded.** internal/cachestore/cache.go:199 increments generation[account] on every successful mutation, but neither LRU eviction nor expiration removes account generations. maxEntries bounds items, not this map. Memory therefore grows with every distinct account that writes. Prefer a globally monotonic invalidation epoch (safe but more conservative for in-flight fills), or bounded/refcounted account state. Do not simply delete generations: a stale in-flight fill can otherwise become valid again. Add an account-churn plus in-flight-fill regression.

2. **P2 — Invalid dates/months reach storage from the domain API.** internal/journal/service.go:39/53/128/165 checks future ordering but never validates Date.IsValid or month/year range. A temporary probe saved Year=2020, Month=13 through Service+local Store successfully; Month(2026,13) also succeeded. Google Put instead returns ErrInvalidText for an invalid date (internal/provider/google/store.go:270), incorrectly coupling date failures to text errors. Validate at the domain boundary with a dedicated date error; reject before storage access and align HTTP mapping. Add invalid/zero dates and month/year bounds to domain/shared contracts. HTTP parsing currently blocks most of these calls, so this is a domain-contract gap rather than a demonstrated normal-editor failure.

3. **P2 — Early years cannot round-trip through paths.** internal/journal/types.go:78 formats the directory using %d while ParseEntryDate requires four digits. For year 0123, EntryPath emits entries/123/0123-01-01.txt, which parsing rejects. Local Put/Get can succeed while ListYear omits the saved entry; Google's synthetic path construction has the same discrepancy. Standardize a supported year policy and four-digit directory formatting across adapters, then test 0001/0123/0999/1000/9999. Preserve existing files when deciding migration behavior.

4. **P2 — Configured logging is not used by request logs.** cmd/app/wire.go:newLogger builds a configured logger, but cmd/app/main.go:42 uses it only for startup. internal/web/middleware/request_log.go:18 and main's failure logging use slog.Default; nothing installs the configured logger as default. LOG_LEVEL=error therefore does not suppress middleware Info requests. Inject the logger into middleware or install the configured default deliberately at startup. Test level filtering without leaking request data.

5. **P2 — Settings silently drops provider failures.** internal/web/router.go:112 ignores FolderURL errors, while internal/web/handler/settings.go propagates them. The HTML page can present an absent folder as though it simply does not exist after an auth failure or outage. Preserve a safe unavailable/reconnect state and retry path, without exposing raw provider errors. Share the Settings lookup orchestration between its two delivery paths where practical.

6. **P2 — Template errors can be reported after success is committed.** internal/web/render.go:59 executes directly to the response and then calls http.Error on failure; partial HTML may already have committed 200. renderNotFoundPage also writes 404 before renderPage sets Content-Type. Render into a small buffer before committing headers/status; accept the intended status explicitly. Test failure and styled-404 content type. Embedded templates lower the likelihood, but the error handling is not reliable.

7. **P2 — Local cancellation is inconsistent.** internal/provider/local/store.go:35/56 ignores ctx in ListYears/ListYear; Put/Trash wait on a process-wide Mutex before checking cancellation. A canceled request can perform a full directory scan or wait behind another account/root. Check cancellation before and during scans and make lock waiting cancelable, retaining shared bounded mutation coordination and atomic replacement. Previews also reads whole files while Google uses a bounded prefix; improve that only with tests preserving UTF-8 and leading-whitespace behavior.

## Maintainability and architecture cleanup

8. **Shared provider IDs and named limits.** The example in internal/web/handler/auth.go:24/60 is valid: local/google literals are repeated in registry wiring, config, routing, credentials, and providers. Define untyped LocalID/GoogleID constants in internal/provider/provider.go, which all relevant layers can import inward. Use generated api.StartProviderLoginParamsProviderLocal etc. only for API-typed comparisons. Never import internal/web/api into providers/config to reuse generated constants. Name policy values near their owner: encoded cookie and callback limits, credential limits, token-cache/lock capacities, preview length/budget, retry policy, and provider timeout. Use net/http status constants in Google error mapping. Keep distinct limits distinct; 4096 for a code and 4096 for a cookie need separate names. Ordinary zero values, empty versions, UTF-16 ranges, and protocol field names do not all need a shared constants package.

9. **Configuration belongs at startup.** internal/provider/local/local.go:21 guesses the meaning of variadic strings and re-reads environment/defaults; internal/provider/google/google.go:27 also loads environment independently of internal/config. cmd/app/wire.go parses and validates provider names again and uses a local-adapter URL helper for Google test-mode policy. Load/validate configuration once, pass explicit constructor inputs, and own generic loopback policy in configuration or the composition root. Avoid a new generic utilities package. Keep Wire generation authoritative.

10. **Session policy is duplicated across HTML/API.** internal/web/router.go:185 and internal/web/handler/server.go:35 each decode/check age/provider availability. Put shared request-session reading in one small web/session integration component with presentation-specific redirects/JSON outside it. Centralize authenticated-cookie construction as well so login/clearing cannot drift on Path, Secure, SameSite, or lifetime. Keep core session encoding independent of HTTP.

11. **Auth validation needs readable stages and safe diagnostics.** internal/provider/google/auth.go:55 and internal/web/handler/auth.go:154 combine many rejection causes in long boolean expressions. Extract small focused checks, sample now once per validation, and retain constant-time state/nonce comparison. Use bounded internal reason categories (transaction, exchange, claims, grant, account, refresh) for safe diagnostics, while keeping generic public errors. Never log OAuth query values, token payloads, raw external errors, or account claims. This would make the recorded unexplained callback rejection diagnosable; it does not establish that failure's cause.

12. **Avoid repeated Drive discovery in previews.** internal/provider/google/store.go:397 calls entryFile for each date; that discovers root, year folder, and file before each download. A cold 20-item page can require roughly 60 list requests plus 20 downloads in addition to index loading. Four workers limit concurrency but not round trips. Resolve root/year once per operation and group file lookup by year before bounded downloads; verify request counts with mocks. Measure before adding long-lived caches, which risk rename/move/external-change staleness.

13. **Small Go clarity issues.** Flatten cache ListYears/ListYear's returning if/else branches; give Cache.New an explicit capacity option rather than a variadic argument that silently ignores extras; fix token-cache replacement at capacity so refreshing an existing key does not unnecessarily evict another entry (auth.go:119). Use named fields for credentials literals, consistent stdlib/external/internal import groups, useful godoc for exported constructors/functions, and normal separation between functions in Google files. Add compile-time Store/Provider assertions where absent. Do not create interfaces just to satisfy a style rule or replace clear concrete constructors with unnecessary factories.

14. **Tests and lint should cover these risks, not every literal.** Add targeted regressions for findings 1–7 and token-cache replacement. Add shared provider-ID consistency coverage at API/config boundaries while keeping some independent expected wire strings so tests can detect accidental contract changes. Current lint catches correctness basics but not constant duplication, API docs, or long functions. Introduce selected goconst/revive/complexity checks gradually with reviewed exclusions for generated code, fixtures, and spikes; avoid enabling every stylistic linter at once.

## Existing production gate, not a newly discovered regression

Drive Put checks md5 then uploads (store.go:302–324); the optional ETag is not a verified atomic precondition. External writers can still race, and process locks do not coordinate replicas. Keep production disabled until the policy is resolved. A constants refactor cannot fix this architectural limitation.

## What to preserve

The existing journal/Store/Clock direction, explicit composition root, narrow Drive scope, encrypted cookies, safe text handling, atomic local saves, account-isolated caches, and tested draft/conflict/autosave behavior should remain. cloud.google.com/go/civil is a plain date value dependency, not an accidental Google API dependency. Domain and use cases can remain together in journal; no directory-layer rewrite is needed. OpenAPI and Wire files remain generated from their sources. Independent spikes can retain experimental helpers; do not make production depend on them or refactor historical experiments solely for uniformity.

## Recommended sequence

1. Correct cache bounds, domain date/path contracts, and logger wiring with focused tests.
2. Introduce provider IDs/named policy constants and explicit startup configuration; regenerate Wire when changed.
3. Consolidate request-session and Settings orchestration; improve buffered rendering and cancellation.
4. Simplify auth validation and add privacy-safe diagnostics; optimize measured Drive round trips.
5. Strengthen selected lint rules, then resume remaining Google reliability/release acceptance.

This report is a proposed backlog, not authorization to apply all changes or merge PR #11.

## Implementation follow-up — 2026-10-10

Implemented the correctness and maintainability work in this report: bounded scalar cache invalidation epoch; domain date/month validation and padded paths with legacy local-directory compatibility; configured default logging; visible Settings folder failure/retry; buffered template responses; cancellation-aware scans/lock acquisition; shared provider IDs and owned policy limits; explicit startup configuration and regenerated Wire; shared web session/cookie policy; staged auth validation with allowlisted diagnostics; batched Drive preview discovery; token replacement at capacity; adapter assertions and selected revive linting. Durable conventions are recorded in AGENTS.md.

Focused regressions cover invalid dates/months, early-year paths, legacy local files, canceled scans/lock acquisition, stale cache fills after account churn, token replacement, rendering failures/404 headers, folder-error presentation, session policy, log-level filtering, and preview list-request counts. Local previews still read whole files; bounded-prefix local previews remain an optional performance improvement rather than part of this refactor. No architecture-layer directory rewrite or new generic utilities package was introduced. Historical report line numbers refer to the reviewed snapshot.

Production Google remains disabled; these changes do not solve the external-writer checksum/upload race or explain the historical callback rejection. Browser/live Google acceptance has not been repeated for this refactor.
