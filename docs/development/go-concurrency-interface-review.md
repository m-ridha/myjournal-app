# Go concurrency and interface review (2026-10-12)

Scope: journal/connection use cases, provider/cache/session adapters, web handlers/router, composition root and their test seams. This is static design triage with regression tests, not an exhaustive security audit or live latency measurement.

## Implemented

- Renamed `journal.Service.Page` to `ListEntries` throughout handwritten code/tests and benchmarks. Its bounded API pagination behavior is retained. Browser Entries always requests page 1/size 20, removes pagination controls and points to Calendar for older notes. Both locales describe the limit.
- `handler.JournalService` owns the five use-case methods consumed by journal HTTP handlers. `ServiceFactory` returns this interface; core `NewService` still returns `*journal.Service`. Wire was regenerated. A handler test supplies a stub without constructing a journal service or accessing journal storage, verifying response/error mapping and forwarded pagination parameters.
- Confirmed the early-exit behavior with existing and new tests: the newest year's twenty returned entries plus a lookahead stop traversal before an older year when 21 dates are available there. Previews cover only twenty dates. Each visited year still loads its complete metadata index. If twenty dates exhaust that year, lookahead may require visiting an older year even though the browser hides navigation.

## Concurrency triage

| Path | Current behavior and recommendation |
| --- | --- |
| `internal/core/journal/service.go` | `ListYears` precedes year selection; preview dates depend on selected metadata. Sequential year traversal avoids unnecessary older-year requests. Do not eagerly load all years. Bounded lookahead is a measured candidate for sparse multi-year pages, rather than an automatic improvement. |
| `internal/driven/cache/journal/cache.go` | Four concurrent preview-date loads, batch/date singleflight and bounded admission already exist. Compare four versus eight with cold/warm workloads, request counts and throttling before raising limits. Hot cache hits do not establish cold Drive latency. |
| `internal/driven/provider/google/store.go` | Direct batch previews already use four workers; decorated calls use singleton batches, so at most four content downloads run per decorator batch, rather than sixteen. Starting up to `min(4, len(dates))` adapter workers could reduce goroutine overhead, but is not a demonstrated network-latency improvement. Independent multi-year preview discovery is another candidate only for actual multi-year batches. |
| `internal/driven/provider/google/discovery.go` | Operation-local singleflight already avoids holding the metadata mutex across I/O. Keep generation-safe clears and fresh upload checks. |
| `internal/driven/provider/google/auth.go` | Refresh is serialized/rechecked by grant. Exchange, validated identity and account-bound credentials are dependent steps. Preserve this ordering; do not duplicate refresh work to seek parallelism. |
| Google/local mutation paths | Account/file mutation ordering and fresh conditional checks protect data. Do not parallelize check/upload or folder creation. Drive list pagination needs each preceding page token. |
| `internal/core/connection/service.go`, web sign-out/disconnect | Revoke before clearing cookies/drafts is a safety requirement, not a concurrency opportunity. |
| Local storage, codec, configuration, locale/rendering | Mostly filesystem/CPU/startup work. The shared decorator already parallelizes local previews. No measured bottleneck justifies adding goroutines to small parsing, encryption or rendering functions. |

## Interface standardization triage

- Keep concrete returns from constructors for providers, renderers, registries, services and cache adapters. This remains idiomatic and does not prevent interface-based injection at consumers.
- Existing core `Store`, `Clock`, connection `Gateway`/`Catalog`, and cache-owned `ByteCache` already describe actual boundaries. Preserve them; no parallel abstraction hierarchy is needed.
- Web still consumes concrete connection services (`Begin`, `Complete`, `Disconnect`, `Settings`). A consumer-owned connection interface would make isolated auth/settings handler tests easier, analogous to the journal change. It is a legitimate follow-up when those tests need direct workflow stubs; existing core tests already fake gateways. No blanket interface conversion was applied.
- Router/rendering and registry use do not currently demonstrate a replacement implementation or isolated test need. Splitting the five-method journal handler contract into many tiny interfaces would add composition without reducing this factory's actual required method set.

No extra goroutines, broader Drive scopes, new API endpoints or speculative provider abstractions were introduced by this review. Live Drive timing, limits of four versus eight, and sparse cross-year benchmarks remain unmeasured candidates.

## Verification

Fresh `make fmt lint test` passed with zero lint issues, all Go packages and six JavaScript suites. Focused race checks covered core journal, web packages, cache policy, Google and the composition root. Browser regressions verify fixed page/size despite URL parameters, bounded rendering and Retry. Chrome with 25 temporary synthetic entries confirmed exactly the latest twenty, no pagination controls, Calendar access to an older saved date, Indonesian text and no horizontal overflow at 390px. Owned servers and temporary data were cleaned up.

The accessibility script initially found the new inline Calendar link distinguishable only by color. Extending the existing persistent-underline style to lead-text links fixed this; the fresh rerun passed 34 axe scans and keyboard checks with zero violations. Axe retains incomplete checks; this is not full WCAG certification or native screen-reader verification. Reports: `/private/tmp/myjournal-recent-a11y.json` and `/private/tmp/myjournal-recent-browser.log`. No live Drive measurement, release or deployment was performed.

## Follow-up superseding worker-limit candidates

The cold Entries follow-up in [performance measurements](performance-benchmark.md#cold-entries-follow-up-2026-10-12) implements operation-local year-folder query reuse, eight cache preview workers and `min(4, len(dates))` Google workers. The four-worker observations above record the earlier review state. Live Drive latency/throttling remains unmeasured. The inline Calendar link was subsequently removed by user request; header navigation remains.
