# Architecture and decisions

This guide owns current architecture decisions and their rationale. Product behavior is in [requirements](requirements.md); operational procedures are in [operations](operations.md). Git history retains superseded designs and investigations.

## Stack and boundaries

Use Go, chi, `html/template`, embedded assets, Google Wire and oapi-codegen. The module is `myjournal-app`; preserve the Go version in `go.mod`. JSON and vanilla JavaScript are an explicit choice: no HTMX, frontend framework or Hugo build/runtime dependency.

| Location | Responsibility |
| --- | --- |
| `cmd/app/` | Startup configuration, provider registration and Wire composition |
| `internal/core/journal/` | Journal use cases, Store/Clock ports and shared store contracts |
| `internal/core/connection/` | Connection models, lifecycle, ports, registry and application service |
| `internal/driven/provider/{local,google}/` | Development filesystem and Google OAuth/Drive adapters |
| `internal/driven/cache/cachestore/` | Journal caching policy and consumer-owned ByteCache port |
| `internal/driven/cache/local/` | Interchangeable BigCache and LRU byte storage; BigCache is wired |
| `internal/driven/clock/`, `internal/driven/session/` | System clock and authenticated cookie encryption |
| `internal/config/` | Environment parsing and validation |
| `internal/driver/web/` | Pages, thin API handlers, middleware, locale catalog and generated API |
| `assets/`, `tests/` | Embedded templates/static assets and browser regression suites |

Dependencies point inward. Core imports no HTTP, rendering, OAuth, Google APIs, filesystem or database packages; providers do not import web. Only `cmd/app` composes concrete providers, clocks, caches and the cookie codec in production. Driving/driven integration tests may compose adapters. Depguard enforces these boundaries; the architecture check exercises forbidden imports, including nested packages.

`journal.Service` owns open/save/delete/list/calendar. Store and Clock stay in `internal/core/journal/store.go`; versions are opaque. Handlers use their consumer-owned JournalService interface and an injected session-scoped factory. `connection.Service` owns login, completion, disconnect and settings. HTTP adapts cookies and responses through injected authsession.Codec and handler.ConnectionService contracts. Gateway.RequiresLoginFlow expresses pending-login capability independently of provider IDs. Core constructors return concrete services; introduce interfaces at actual boundaries.

## Storage and concurrency

Google files are plain UTF-8/LF text at `myjournal/entries/<year>/YYYY-MM-DD.txt`. Private `appProperties` identify folders, so renaming does not break discovery. Reads never create folders; writes recreate missing paths. Select duplicates deterministically without deleting or merging data-bearing folders.

Drive md5Checksum is the content version. Validate downloaded text, size and checksum/version agreement; absent checksums fail explicitly. A pre-upload comparison followed by upload is **not atomic**. Optional metadata ETag forwarding has no verified external-writer atomicity guarantee. Shared account locks coordinate this process only; deployments can overlap, and duplicate new-date files or lost updates remain possible across processes or external writers.

The local adapter uses atomic replacement, context-aware same-process coordination across stores, UTF-8 validation and collision-safe trash preservation. It is development-only and does not coordinate unrelated external processes. Every adapter must pass shared Store contracts. Pagination validates pages 1–10000 and sizes 1–100 before storage access and retains at most size+1 dates; only requested-page previews are loaded.

## Sessions and authentication

Production has no user database or server-side session store. AES-GCM encrypts/authenticates versioned connection cookies; purpose binding separates the login cookie. Session lifetime is fixed at 30 days from original IssuedAt, enforced by server age and cookie expiry. Refresh never extends it; shorter future policies must also constrain existing cookies. Cookies carry refresh credentials rather than access/ID tokens; the access-token cache is bounded.

A separate ten-minute pending-login cookie holds state, PKCE and nonce. Validate callback shape, provider binding, age and state before exchanging a code; validate ID-token signature, claims and nonce. There is one pending login per browser. Stateless cookie deletion is not a replay ledger; code single use and PKCE remain necessary. Preserve an omitted refresh token only for a validated same-account reconnect; reject unexpected account switches.

Only request `openid`, `email`, `profile` and `https://www.googleapis.com/auth/drive.file`. Local providers and GOOGLE_TEST_MODE require exact localhost/loopback origins; use a separate Desktop app OAuth client for this—Desktop app clients permit loopback redirects by design without triggering App Security warnings, and do not share credentials with the production Web application client (see [operations](../development/operations.md)). Ordinary Google startup requires HTTPS and Secure cookies. Configuration rejects invalid session keys and unknown/duplicate providers.

## Bounded caching

The journal decorator owns policy through ByteCache; storage adapters do not know journal semantics. Provider/account-isolated full-entry and confirmed-absence snapshots have a two-minute TTL, with existence, text and version retained together. Other errors are not cached. Previews derive from those snapshots. Unrelated snapshots retain their original expiry after mutations. Logical expiry/eviction does not promise immediate physical byte erasure; the BigCache byte-queue setting is not a total process-memory bound.

Operation-scoped discovery starts fresh for each service call; decorators forward its scope. Google retains at most 16 discovery queries and 1000 files per query; larger operations can repeat discovery. Shared singleflight reads use generation/provider/account/parameter keys, bounded admission and request-derived deadlines. Batch and date coordinators together admit at most 128 distinct keys and 640 registrations. Returned maps/slices are copied; credentials do not enter coordination keys. Authentication-failure followers can retry through their own validated Store. Mutations and pre-upload checks never join read flights.

OpenFresh, selected by HTTP `Cache-Control: no-store`, bypasses retained snapshots for recovery, failed-save reconciliation and conflict actions. Fresh reads use separate flights. Ordinary opens can see content up to two minutes old, including changes from another process. Mutation attempts invalidate affected snapshots with generation guards, even after ambiguous failures; never seed content from save acknowledgements.

Successful PUT/DELETE may schedule backend read-through warming: four workers, at most 64 active/queued dates, ten-second deadlines, fresh provider clients and operation scopes. Repeated dates coalesce. Overload skips optional warming; failures do not undo successful writes. Creation and trash warm ListYears and the affected ListYear before the entry snapshot; existing-entry updates warm only the entry. Warming never changes editor/draft state or retries writes. Shutdown cancels and joins workers before closing byte storage.

## HTTP and generated contracts

Browser pages and `/api/v1/...` JSON APIs are separate; `/healthz` is unversioned. The authoritative API schema is [OpenAPI](../api/openapi.yml). Preserve explicit save strings, JSON error envelopes and wrapped domain error mappings via errors.Is. Request limits include a 1 MiB save envelope. SameSite cookies do not replace net/http.CrossOriginProtection for unsafe browser requests or OAuth state/PKCE.

Use same-origin CSP and local external scripts/assets; no inline script exceptions, trackers, fonts or CDNs. Escape text through templates and safe DOM APIs. HSTS is one year, host-only, for the configured HTTPS origin, without subdomains/preload; never infer it from untrusted forwarded headers. Logs contain method, query-free path, status, duration and request ID, without raw sensitive errors or callback queries.

HTTP lifecycle uses bounded header/read/write/idle and handler deadlines, graceful draining on shutdown, then cache cleanup. Preserve cancellation through external operations and lock waits. Generated API, Wire and Mockery output remains committed and generator-owned; edit the owning schema/input and regenerate.

## Visual provenance

The interface is custom Go templates/CSS inspired by [Hugo Monochrome](https://themes.gohugo.io/themes/hugo-theme-monochrome/). A recorded comparison against upstream commit `98923c5fda5a4a6755adc6f03721f6d8db4dd060` detected no copied source; this does not prove every historical snippet's provenance. The upstream MIT license identifies copyright 2021 kaiiiz; no upstream theme version is bundled. Preserve the project's existing [license](../../LICENSE).

The trash icon's user-supplied source is [SVG Repo #335255](https://www.svgrepo.com/svg/335255/trash), whose page listed CC0 when reviewed. Direct download comparison was unavailable, so no independent byte/geometry match is claimed. Icons remain separate SVG assets using currentColor and accessible labels; the PNG favicon has separate provenance from the theme comparison.

## Public search discovery

Only the informational home, Help, Privacy and Terms pages have canonical metadata and sitemap entries. Each has deterministic `/en/…` and `/id/…` URLs; explicit locale paths override the preference cookie and save that language on GET. Unprefixed URLs remain compatible and canonicalize to the corresponding locale version. The authenticated `/` still renders the editor with `noindex`; explicit language home pages remain informational and offer the editor link when signed in. Language-switch POST redirects accept only the existing browser allowlist and explicit public variants.

Canonical URLs, reciprocal hreflang links, share images and sitemap locations derive from configured APP_BASE_URL and a fixed public-page allowlist, never request hosts or queries. `x-default` points to the unprefixed page. WebSite microdata identifies the site on public home pages without inline scripts or CSP exceptions. Authentication, settings, journal and error pages have `noindex`; authentication remains the security boundary. Robots only allows crawling and advertises the sitemap, without enumerating private routes or directories.

Signed-out informational pages load only theme/language controls and the small shared cross-tab draft cleanup listener. They carry only theme translations in their script payload. Editor/application code remains available on authenticated pages and Settings; calendar-picker code loads only on Calendar. Date correction runs only where dates are used (editor and Calendar). Keep `no-store` for responses that may vary by session or set a preference cookie.
