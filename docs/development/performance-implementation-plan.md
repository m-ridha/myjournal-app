# Measured performance improvements — implementation plan

Status: focused implementation and initial validation complete; stronger live evidence is optional and production gates remain open.

Date: 2026-10-10. Implements the user-approved sequence from [baseline findings](performance-benchmark.md), using inline execution without delegation. Preserve existing working-tree edits and do not commit/push without a further request.

Architecture: retain journal/Store/Clock ports and inward dependencies. Add an optional operation-scope capability at the Store boundary, forwarded through the cache. Google query reuse exists only in the returned operation store, never on the long-lived adapter. Keep process locks, fresh editor reads, fresh pre-upload checks, and production guards.

- [x] Upload candidate: failing regression for bounded upload allocation/body integrity; use SDK single-request streaming option if it preserves no ambiguous-write retries. Compare six repetitions with the original buffer baseline.
- [x] Discovery candidate: failing request-count and scope-lifetime regressions; introduce optional Store operation scope, create it at each Service operation, forward it through cache, and bound query reuse within that scope. Invalidate scope metadata after folder creation. Compare cold-page counts and create behavior; preserve duplicate/move/trash behavior.
- [x] Invalidation candidate: failing tests for unrelated previews/index retention, changed-date invalidation on unchanged acknowledgement, create/trash membership refresh, TTL and stale concurrent fills. Keep the scalar epoch; invalidate target previews, preserve each index independently only where its unexpired cached membership establishes an update, otherwise invalidate affected-year/all-years indexes. Never extend retained TTL or seed acknowledged text into preview caches. Preserve failed/ambiguous mutation safeguards.
- [x] Compare after-save visible and off-page dates separately. Retain raw baseline and per-candidate output, source hashes and limitations; do not confuse local HTTP timing with live Drive latency.
- [x] Run make fmt lint test, separate architecture probes, focused race tests and vet; update plan, progress/findings and live comparison instructions. No automatic fixture retry or cleanup. Later read-only January 3 inspection confirmed final attempted content is present; root-cause investigation was deferred by the user.

Deferred: larger CLI/fixture programme, Range reads, persistent ID hints, transport tuning, broad concurrency load, and production enablement. Live benchmark date/progress guard is already fixed; do not run another full live probe to diagnose the retained January 3 failure.

- [x] User-assisted targeted live comparison recorded: post-save browsing ~66% lower median; changed saves show no improvement. Exact build identity was not independently verified.

- [x] Capture January 5 safe live save timings; upstream calls account for 99.94% of aggregate server duration. No visible retries in ten measured successful saves.
- [x] Keep all three candidates with documented resource/TTL/concurrency tradeoffs; implement explicit uncertain-save retry and validate it in Chrome with the local provider.
- [x] Update progress/findings and retain raw evidence; no production enabling implied.

Optional follow-up: repeat controlled live comparisons in another session, recording exact running build and cache/auth state. This iteration does not establish reliable tail latency or failure-rate acceptance. Full CLI/large fixtures, field-mask/Range experiments, persistent hints and transport/concurrency tuning remain deferred. External-writer policy and remaining Google acceptance are separate production gates. January 3 root-cause work is no longer an active task.
