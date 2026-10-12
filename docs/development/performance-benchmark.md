# Performance baseline

## Scope and commands

Measure before changing performance-sensitive production code. Start with request counts and local overhead; report live API latency separately. Do not interpret fake Drive timings as Google latency or release acceptance.

```sh
go test ./internal/driven/provider/google -run '^$' -bench '^BenchmarkDriveBaseline$' -benchmem -benchtime=20x -count=6
```

The benchmark executes the real journal service, cache, Drive adapter, and generated Google HTTP client against the existing fake Drive HTTP server. It provisions 30 January 2020 entries with identical 1,920-byte synthetic text before timing, using one listing page (fake page size 1000). No credentials, real account, or external network are used. Four preview workers remain enabled. Cold means a new data cache with an already-used local HTTP client; it is not process-cold. Warm browsing is explicitly primed. Editor opens bypass the data cache. Cache capacity is 2048 and TTL is two minutes.

Each scenario runs 20 operations in each of six independent repetitions. Reported ns/op is a repetition's mean, not a per-operation sample. The table below uses the median of those six means, not request p50/p95. Counts include all fixture-operation HTTP requests, including concurrent previews; setup requests are excluded. The after-save scenario times both changed save and subsequent page fetch. Cache construction is included in cold scenarios. Allocations include the fake HTTP server and generated client, so they are not a production heap measurement.

## Synthetic baseline — 2026-10-10

Base application commit: `0c7c5cca86080724c869d9e4e7b76b28730b4ed3`, with uncommitted benchmark-only instrumentation. Go 1.27.1, darwin/arm64, Apple M4. See [manifest](../../benchmarks/results/2026-10-10-synthetic/manifest.json), [raw output](../../benchmarks/results/2026-10-10-synthetic/raw.txt), [parsed repetitions](../../benchmarks/results/2026-10-10-synthetic/summary.json), and [allocation profile](../../benchmarks/results/2026-10-10-synthetic/allocation-profile.txt). The manifest includes source hashes to identify the uncommitted harness.

| Scenario | Median repetition mean (ms/op) | HTTP calls/op | Request breakdown |
| --- | ---: | ---: | --- |
| Existing entry open | 0.655 | 4 | 3 lists + 1 download |
| Changed save | 0.999 | 5 | 3 lists + 1 metadata GET + 1 upload |
| Unchanged save | 0.653 | 4 | 3 lists + 1 metadata GET |
| Entries, data-cold, size 20 | 2.549 | 28 | 8 lists + 20 downloads |
| Entries, warm | 0.0035 | 0 | cache hits |
| Changed save then Entries | 3.486 | 33 | 5 save calls + 28 browse calls |
| Calendar, data-cold | 0.696 | 3 | root/year/files lists |
| Calendar, warm | 0.0025 | 0 | cached year index |

All six repetitions agree on these request counts. All benchmark operations succeeded; errors abort the run rather than disappear from the report. OAuth, refresh, real network latency, rate limits, external changes, and HTTP/browser rendering overhead are excluded. This initial baseline omits creation, trash, missing dates, multi-year pages, and large-entry timings.

## Findings and candidate priorities

1. **Upload buffering is a concrete allocation lead.** Changed saves allocate a median ~17.04 MB/op versus ~106 KB/op for unchanged saves. A separate allocation profile attributes ~98% of whole-run allocated space to the SDK's `gensupport.NewMediaBuffer`; that profile includes untimed fixture creation. The installed google.golang.org/api v0.298.0 uses a default 16 MiB upload chunk. This identifies the allocation source, not a measured production latency impact. Evaluate explicit upload chunk sizing as a separate candidate, preserving body integrity, size bounds, ambiguity/retry behavior, and checksum acknowledgements. No upload behavior has been changed.
2. **Metadata discovery repeats.** Eight list calls precede a cold page's 20 downloads. Request-scoped reuse across index and preview discovery could reduce work, but must have an explicit lifecycle and retain deterministic duplicate selection and external-change behavior.
3. **Successful saves erase browsing warmth.** The combined save/page operation makes 33 calls, even with the page primed before the first save. Account invalidation forces the same 28-call browse path after every save. Targeted invalidation is a later candidate requiring stale-fill and external-change tests; skipping invalidation for unchanged saves without reconciling cached data is unsafe.
4. **Warm cache is effective.** Warm Entries/Calendar make zero fake Drive calls. Do not add Redis or persistent editor-content caches to address latency that has not been measured.

There is no candidate comparison or claimed speedup yet. Before choosing latency work, collect real API samples. A buffering experiment can separately demonstrate allocation reduction with the existing benchmark, but must not be presented as faster Google requests without live evidence.

## Live exploratory probe

Browser automation can operate the app but its read-only page scope does not expose the Performance API or authenticated script execution. Run the same-origin probe manually so HttpOnly session material stays in the browser:

1. Keep other app tabs idle and open `http://localhost:3030/privacy`, signed in with Google. No editor should be active in this tab. Wait for any earlier probe to settle before pasting a new version. The updated probe logs its date/scenario and rejects overlapping invocations, including when pasted again while it is running.
2. Paste [scripts/benchmark-browser.js](../../scripts/benchmark-browser.js) into DevTools Console, then run `await mjBenchmark('2020-01-01')`, substituting an empty past date. Existing entries are refused. This creates one synthetic fixture, runs ten samples each for open/changed save/unchanged save/repeated Entries/repeated Calendar/Entries after save, then trashes the fixture and verifies absence. An error stops further writes; inspect any retained fixture before manual cleanup. Do not run concurrently or retry an ambiguous failure blindly.
3. Save only the returned timing/status result. Never export cookies, callback parameters, account values, versions, or journal text. The probe returns none of those values.

Timings include full API response-body consumption but exclude rendering, pacing, setup, cleanup, and (for Entries-after-save) the preceding save. The session/token/connection path is primed outside timing. Browsing cache state and token cache hits are not instrumented in this probe, so repeated calls must not be called verified warm/cold. The first browsing sample may differ from later ones; retain every sample. The user's actual entry count and page contents influence browsing cost, so this is exploratory rather than the controlled 30-entry fixture comparison. Record browser/build/network separately. Ten samples do not establish reliable tail percentiles.

## Live exploratory results — 2026-10-10

User-provided [raw browser samples](../../benchmarks/results/2026-10-10-live-exploratory/raw.json) contain 60 measured requests, all HTTP 200, with `cleanupVerified: true`. Medians were independently recomputed and agree with the returned summary. See [manifest](../../benchmarks/results/2026-10-10-live-exploratory/manifest.json) for unknown build/browser/network details. The attachment says January 1; the user initially associated its appearance with a January 2 invocation. Later console history shows an earlier pending invocation and earlier-date results appearing after newer commands, consistent with overlapping asynchronous runs. Keep raw samples associated with their own result date; exact overlap timing is not established. A subsequent existing-entry refusal on January 1 proves the overwrite guard worked, not the cause or ownership of the currently present entry. Do not infer current fixture cleanup from the historical success result.

| Scenario | Samples | Median | Min–max |
| --- | ---: | ---: | ---: |
| Open existing entry | 10 | 2.270 s | 2.129–2.508 s |
| Changed save | 10 | 3.333 s | 3.158–5.183 s |
| Unchanged save | 10 | 1.815 s | 1.603–3.633 s |
| Entries, repeated | 10 | 4.55 ms | 2.90 ms–7.074 s |
| Calendar, repeated | 10 | 4.95 ms | 2.70–6.40 ms |
| Entries after changed save (browse only) | 10 | 7.078 s | 6.601–7.842 s |

The first Entries request took 7.074 seconds; subsequent nine took 2.90–4.90 ms. Calendar followed Entries, which had already populated year indexes; this is not an independent cold Calendar measurement. Every post-save Entries request took over six seconds. These observations strongly support post-save cache invalidation/repopulation as a user-visible bottleneck, consistent with synthetic request-count results. Live cache hits, Drive attempts, retries, and per-call timings were not instrumented, so this is supporting evidence rather than a direct attribution of each second to a specific call.

Open and save paths remain slow even with primed authentication: sequential discovery is a plausible contributor based on source and synthetic counts. Network/upstream variability and retries could also contribute; no live breakdown distinguishes them. The 16 MiB upload-buffer allocation is independently established locally but is not established as the reason for the live save latency. Unchanged saves avoid upload yet still take a median 1.815 seconds. Browser autosave normally suppresses unchanged content, so this manual API scenario is not a claim of routine redundant autosaves.

Next candidate priority: reduce repeated metadata discovery with explicit operation-scoped reuse, then evaluate safe targeted invalidation against this post-save browse scenario. Upload-buffer sizing is a separate small allocation experiment. No candidate is implemented or speedup claimed. Ten samples from one session do not establish tail latency or production acceptance. No further run is necessary merely to resolve the date label.

## Verification

`make fmt lint test` passes, including five JavaScript suites (the probe tests cover refusal to overwrite existing entries, stopping on failed writes, sanitized output, and verified cleanup). The benchmark scenarios also pass under the race detector with `-benchtime=1x`. All seven architecture probes passed separately with cleanup; keep these probes separate from lint/build. Keep baseline/candidate instrumentation identical and retain raw results before any optimization.

## Additional live evidence and probe corrections — 2026-10-10

The next [successful attachment](../../benchmarks/results/2026-10-10-live-jan02/raw.json) labels its own date January 2 and reports sixty HTTP-200 samples with cleanup verified. Medians: open 2.206 s, changed save 3.385 s, unchanged save 1.717 s, repeated Entries 4.05 ms (first sample 6.965 s), Calendar 4.45 ms, and post-save Entries 6.673 s. These corroborate the browsing pattern, but are not independent controlled sessions; console history suggests overlapping invocations may have occurred. Do not pool runs or infer a precise cache-hit/latency attribution.

The [January 3 failed result](../../benchmarks/results/2026-10-10-live-jan03-failed/raw.json) contains ten successful opens (median 2.128 s), nine successful changed saves, and a tenth changed save returning HTTP 500 after 21.743 seconds. Save median including the failed attempt is 4.309 s; successful-only median is 3.442 s. Keep the failure in results. The probe correctly stopped and did not clean up. The upload may have succeeded without acknowledgement; its current state is unknown. No automatic retry, deletion, or new benchmark run was performed.

The duration is consistent with the configured 20-second external-client timeout plus preceding discovery work, but does not establish the failing operation or cause. The user supplied the generic HTTP-500 JSON message “Something went wrong. Please try again.” at 10:11:02 WIB; terminal duration/backend error category remains unavailable. The message matches handler fallback but does not identify its cause. Source review identified a separate reproducible classification gap: wrapped context.DeadlineExceeded fell through to HTTP 500. A regression reproduced that behavior; the handler now maps it to the existing 503 unavailable envelope. This neither retries ambiguous writes nor increases timeouts, and it does not prove the live failure was a deadline error.

The browser probe now logs the fixture date and each scenario, and holds a shared running guard released even on setup failure. Tests verify rejection of overlapping runs and guard release. Existing refusal, sanitized-report, failed-write stop, and successful cleanup tests still pass. Fresh make fmt lint test passes. These changes improve evidence collection/error classification, not performance.

## Candidate comparisons — 2026-10-10

Implemented the approved sequence from [implementation plan](performance-implementation-plan.md), with six 20-operation repetitions after each candidate. Raw outputs and manifests:

- [Upload streaming](../../benchmarks/results/2026-10-10-upload-streaming/raw.txt): removes SDK chunk buffering, preserving single-request writes.
- [Operation-scoped discovery](../../benchmarks/results/2026-10-10-scoped-discovery/raw.txt): adds bounded exact-query reuse within each use-case call.
- [Visible-date baseline](../../benchmarks/results/2026-10-10-scoped-visible-baseline/raw.txt): same code as the scope candidate, adding a ninth scenario that saves a visible date.
- [Initial targeted invalidation](../../benchmarks/results/2026-10-10-targeted-invalidation/raw.txt) and [final targeted invalidation](../../benchmarks/results/2026-10-10-targeted-invalidation-final/raw.txt): preserve unrelated snapshots and independently reconcile year/all-years membership. The final manifest records production/harness source hashes.

These runs use local fake HTTP, not real Drive. They are sequential repetitions, not randomized paired sessions; benchstat is unavailable. Timings are descriptive medians of repetition means, not statistical significance or per-request tail latency. Counts are identical across repetitions.

| Measure | Original baseline | Streaming only | + Scoped discovery | + Final targeted invalidation |
| --- | ---: | ---: | ---: | ---: |
| Changed-save allocated bytes/op | 17,042,855 | 176,814 | 169,582 | 171,347 |
| Changed-save HTTP calls/op | 5 | 5 | 5 | 5 |
| Cold Entries HTTP calls/op | 28 | 28 | 24 | 24 |
| Off-page save then Entries HTTP calls/op | 33 | 33 | 29 | 5 |
| Changed-save local mean median | 0.999 ms | 0.630 ms | 0.621 ms | 0.628 ms |
| Cold Entries local mean median | 2.549 ms | 2.475 ms | 1.750 ms | 1.803 ms |
| Off-page save then Entries local mean median | 3.486 ms | 3.265 ms | 2.415 ms | 0.655 ms |

The original after-save scenario writes January 1, outside the page's twenty newest January dates. The new visible-date scenario writes January 30. Compare that scenario against its matching scoped baseline: 29 → 9 HTTP calls per combined save/page operation (5 save calls + 4 browse calls), and 2.403 → 1.384 ms median repetition mean. Off-page browsing makes zero calls when indexes have the required unexpired membership; visible-date browsing refreshes only that preview. Do not claim the original 33-call baseline directly measured the later-added visible-date scenario.

Allocation falls by about 99% for small changed saves. No new claim is made about real Drive latency. Warm Entries/Calendar still make zero calls; editor opens still make four and changed saves five. Scope reuse intentionally does not cache editor downloads or pre-upload metadata checks. Cold creation in an existing year can reuse discovery, but creation/trash were not added to the timing matrix.

Regression coverage includes exact text/checksum round trips at the 1 MiB cap, oversized upload rejection before external writes, no ambiguous-upload retry, bounded query retention, no cross-operation reuse, external trash visibility, independent index membership, unchanged-save preview refresh, TTL preservation, creation/trash index refresh, ambiguous-save invalidation, and racing stale fills. See [ADR 015](adr/015-operation-scoped-discovery.md) for lifetime and cache tradeoffs. Retained index ModifiedTime remains a TTL snapshot; current public browsing uses membership, not fresh modification-time ordering.

Keep these candidates for targeted live comparison: allocation and request reductions are established locally; latency and failure-rate acceptance remains open. No persistent identifier cache, transport tuning, Range request, broader OAuth scope, timeout extension, or production enabling was introduced.

### Targeted live comparison

Restart the app with the same `make run` configuration after applying the latest code. Wait for every older probe to finish, then paste the updated browser script on `/privacy`. Use an empty past date (not the retained January 3 fixture):

```js
await mjBenchmark('2020-01-04', {
  scenarios: ['changed-save', 'entries-repeat', 'entries-after-save']
})
```

This runs thirty timed requests plus setup, untimed prerequisite saves, and verified recoverable cleanup. Date/scenario progress and the running guard remain enabled. The original six-scenario invocation still works; targeted options reject empty, duplicate, or unsupported scenarios. Keep all returned samples/failures. Compare with the original matching scenarios, label the current fixture's visible/off-page status where known, and do not pool earlier overlapping runs. The user completed this targeted comparison; results follow. No live fixture was created by automation.

### Targeted live result — January 4 fixture

The [user-reported candidate samples](../../benchmarks/results/2026-10-10-live-targeted-candidate/raw.json) contain thirty HTTP-200 requests and report `cleanupVerified: true` for the 2,230-byte January 4 fixture. Progress reached completed. Compare matching scenarios with the January 2 baseline without pooling runs:

| Scenario | January 2 median | Candidate median | Candidate range |
| --- | ---: | ---: | ---: |
| Changed save | 3.385 s | 3.632 s | 3.295–4.118 s |
| Repeated Entries | 4.05 ms | 3.70 ms | 2.80 ms–3.756 s |
| Entries after changed save (browse only) | 6.673 s | 2.277 s | 2.136–2.594 s |

Post-save browsing median was about 66% lower. The first repeated Entries request still took 3.756 seconds; subsequent requests took milliseconds. Changed-save median was about 7.3% higher, so this run demonstrates no save-latency improvement. All ten measured changed saves succeeded, but this small successful run does not establish a reduced failure rate or resolve January 3's ambiguous write.

Keep the three candidates: measured synthetic allocation/request reductions and this exploratory browsing improvement justify their bounded scope, with integrity/race regressions passing. Preserve existing fresh editor/pre-upload reads, TTL limits, ambiguity handling and production guards. Remaining cold browsing/save latency should be investigated with safe request-category timings before further optimization; do not add persistent ID caching or extend timeouts from these results alone.

Limitations: different fixture dates, sequential exploratory runs, earlier possible probe overlap, unverified browsing cache state, no live Drive call/retry tracing, and no independent verification of the running build. The requested restart preceded this report, but exact build identity was not captured. These are descriptive comparisons, not controlled significance, p95, failure-rate or production-release acceptance. January 3 remains untouched with unknown write outcome.

## Save-latency diagnostics and retained-fixture inspection

The Google Drive HTTP client now emits `google_request` at debug level only. Enable `LOG_LEVEL=debug` and restart with `make run`, retaining the same session key/configuration. Normal info-level operation bypasses timing wrappers. Records contain only bounded category (`list`, `metadata`, `download`, `upload`, `other`), HTTP status (0 without a response), bounded outcome, time to response headers and total duration through EOF/error/close. No URLs, IDs, queries, headers, text, account values or raw errors are logged. Each actual transport attempt gets one record; SDK/read retries can produce multiple records. OAuth token acquisition is included in header time because the wrapper surrounds OAuth; it is not separately measured. Body completion includes consumer/decoder time, not pure network latency. A close before EOF is labeled `closed`, not successful consumption.

Keep other app/editor tabs idle. After restart, prime authentication with a read before measuring, or label the first request as potentially including refresh. Run a changed-save-only probe on an empty past date (never January 3), for example `await mjBenchmark('2020-01-05', {scenarios: ['changed-save']})`. Share sanitized `google_request` records from that window and the outer PUT status/duration, plus probe output. Existing query-free request logs delimit sequential saves; there is no per-request correlation across concurrent tabs, so overlapping windows cannot be reliably attributed. Compare list/metadata/upload durations and repeated attempts before proposing another optimization. Never retry an ambiguous failed write automatically.

For January 3, paste [the read-only inspection helper](../../scripts/inspect-benchmark-fixture.js) on `/privacy` and call `await mjInspectBenchmarkFixture()` with other tabs idle. It reads that date twice through the fresh editor API, compares exact known synthetic A/B bodies in browser memory and returns only boolean flags. It makes no PUT or DELETE and returns no checksum/text. Stable B means the failed final attempt's content is currently present; stable A means the preceding acknowledged content is currently present. Neither proves historical commit timing or the failure cause, and an unrelated/changing body requires further human inspection. Absence is reported independently. Retain January 3 until the result has been reviewed; this helper never authorizes cleanup.

Automated tests cover sanitized timing output, preserved error identity, waiting for body completion, disabled logging passthrough, read-only inspection, exact known-fixture classification and rejection of changing/unrelated matches. Live timing breakdown and January 3 inspection remain pending user evidence.

### January 3 read-only inspection follow-up — 2026-10-10

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

## Overlapping Drive reads (2026-10-12)

Controlled mocked Drive tests compare the same concurrent workload with and without the shared journal cache decorator's in-flight coordination. TTL storage is disabled to isolate collapsing. Counts are metadata list requests / content downloads; a short simulated GET delay makes overlap deterministic. Fixtures use synthetic accounts/content, not live Google traffic.

| Workload | Without sharing | With sharing |
| --- | --- | --- |
| Eight editor reads of one date | 24 / 8 | 3 / 1 |
| Eight cold Entries pages, twenty entries each | 32 / 160 | 8 / 20 |
| Eight calendars of one month | 24 / 0 | 3 / 0 |
| Four Entries pages and four calendars | 28 / 80 | 8 / 20 |
| Two overlapping preview batches, two dates each | 4 downloads | 3 downloads |

Sequential editor reads still download again. Real timing can yield less sharing when earlier work completes before another caller arrives. The initial, uncommitted Google-specific coordinator recorded 4 metadata lists for Entries/mixed bursts; the user-selected provider-independent boundary uses 8 in these fixtures because different Store operations no longer share individual Drive folder queries. Content download reductions remain intact. Neither result is a live latency measurement.

Regression coverage exercises provider/account/generation isolation, independent cancellation, OAuth transport recovery after leader cancellation, admission/deadline/error cleanup, preview budgets, post-save/trash/failed-save reads, and repeated concurrent browsing. [ADR 019](adr/019-overlapping-journal-reads.md) records bounds and tradeoffs. No deployment or acceptance of original performance targets is implied.

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
