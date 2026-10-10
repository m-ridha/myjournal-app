# ADR 015: Bound discovery reuse to journal operations

Status: accepted; exploratory live comparison recorded, production acceptance remains separate

Date: 2026-10-10

## Context

The initial performance baseline measured eight metadata-list requests plus twenty downloads for a cold Entries page. Live exploratory post-save browsing took approximately 6.7–7.1 seconds, while repeated browsing took milliseconds. Saves previously invalidated every cached index and preview for the account. Persistent identifier caching would require broader external-move/trash/duplicate reconciliation.

## Decision

Add optional `journal.OperationStore.ForOperation() Store`. Each journal service operation requests its own view; ordinary stores continue to work unchanged. Cache decorators forward the capability while keeping their existing shared account-scoped cache. The Google view reuses exact discovery queries within the operation, with at most sixteen query results of at most one thousand files each. Oversized results and errors are not retained. The view is never shared across use-case calls. Mutation attempts clear reuse, and folder creation clears it immediately. Editor downloads and pre-upload metadata checks remain fresh; no Google types enter the domain.

Use single-request streaming uploads for bounded journal content rather than the SDK's default 16 MiB buffering. Do not add resumable chunks or retry ambiguous uploads. The 1 MiB content cap and checksum acknowledgements remain enforced.

Target cache invalidation by account/date. Always invalidate every preview budget for the mutated date, including unchanged acknowledgements and ambiguous failures. Retain unrelated previews with their original expiry. A successful update with a nonempty base version may retain the unexpired year index only if it contains that date, and the unexpired all-years index only if it contains that year. Check these independently so off-page updates retain known years while externally added years do not stay hidden. Invalidate either index when its membership evidence is missing. Trash and failed mutations conservatively invalidate those indexes. Keep the scalar generation epoch to reject older in-flight fills; do not seed previews from acknowledgements that could complete out of order.

## Consequences

The cache remains a TTL snapshot: unaffected previews and retained index metadata can reflect external changes only after expiry. Retaining data never renews its TTL. Index ModifiedTime is not refreshed by a successful update; current public list/calendar behavior uses date membership. Any future feature needing fresh modification times must refresh/reconcile them explicitly.

No new account-generation map, long-lived folder cache, background writer, or server-side session store is introduced. Account mutation serialization remains process-local. Reads still do not create folders, and duplicate selection remains deterministic. None of these changes resolves the non-atomic external-writer checksum/upload race or permits production Google registration.

Synthetic measurements verify less work and allocation; live latency benefit is pending a targeted comparison. See the performance report and implementation plan for raw outputs, tests, limitations, and deferred experiments.
