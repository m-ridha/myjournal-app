# ADR 020: Two-minute entry content snapshots

Date: 2026-10-13
Status: Accepted by user

## Decision

Cache ordinary full-entry reads through the shared journal decorator using the existing two-minute TTL and bounded byte storage. Retain text, existence and opaque version together. Cache confirmed ErrNotFound as an explicit absence snapshot with the same TTL and invalidation. Other errors are not cached. Provider/account/date isolate snapshots. This supersedes ADR 019's uncached sequential editor-read policy.

Previews derive their byte-limited strings from the same validated full-entry snapshot and ordinary date flight used by Get. There is no separately retained preview content. The decorator uses ten workers; direct Google Previews uses four and the same validated Get path. Identical preview batches retain bounded in-flight orchestration to preserve operation-local metadata reuse (ADR 019).

PUT and DELETE invalidate the affected entry snapshot and therefore its derived previews; unrelated snapshots retain their original expiry. Preserve existing conservative invalidation after failed mutation attempts, because acknowledgements can be ambiguous. Generation guards reject older in-flight fills. Never seed snapshots from save acknowledgements.

Recovery needs actual storage state: core Service.OpenFresh uses the optional consumer-owned FreshEntryStore port, falling back to ordinary Get for uncached providers. The decorator's GetFresh bypasses snapshot reads/writes and uses a separate bounded flight key. HTTP maps Cache-Control: no-store to OpenFresh; responses prohibit browser/intermediary caching. Browser draft recovery, failed-save reconciliation and conflict actions explicitly request fresh reads. Ordinary opens may reuse snapshots.

## Consequences

Warm opens avoid Drive discovery/download; cold opens still require them. Direct Drive edits and writes handled by another process can remain invisible for up to two minutes. Invalidation coordinates only the current process. Existing non-atomic external-writer and deployment-overlap limitations remain. Full journal content now resides in bounded process memory until logical expiry/eviction; lazy cache expiry does not guarantee immediate physical byte erasure. Privacy copy reflects the expanded cache content. No new persistence, scope, Redis dependency, TTL increase or offline mode is introduced.

Cold lists now validate complete files against the existing 1 MiB limit, UTF-8 and checksum/version agreement, including bytes beyond the former 64 KiB preview prefix. Unsupported or inconsistent files fail the list without partial success; they are not cached as empty entries. Google read discovery reuses complete year listings only within an operation; mutation lookup and pre-upload checks remain fresh. Discovery retention limits remain 16 queries and 1000 files per query, with repeated discovery possible above those limits. Large serialized snapshots may exceed a BigCache shard queue and bypass caching; the existing 16 MiB byte-queue configuration is not a total process-memory bound. See [controlled measurements](../performance-benchmark.md#shared-entry-and-preview-content-2026-10-13).

## Mutation-triggered backend warming (2026-10-13)

After a successful PUT or DELETE, invalidate first and schedule an ordinary read-through for that provider/account/date. Only a validated storage read or confirmed absence can repopulate the snapshot; write acknowledgements never seed it. Save/delete responses do not wait for warming. Failed or ambiguous mutations retain conservative invalidation and schedule no warming.

The composition root supplies a fresh provider Store factory. Each attempt owns a new discovery scope and a ten-second background context, including OAuth refresh; it never borrows the completed HTTP request's client/context. The cache owns four workers and at most 64 active/queued dates. Repeated pending dates coalesce, retaining the latest factory; a successful mutation during an active attempt triggers another attempt. Overload skips warming. Factories/credentials are held only for bounded pending work and released afterward; there is no persistent job or session store. Cleanup cancels and joins workers before closing byte storage, after HTTP draining.

Ordinary readers can join the same entry flight; explicit fresh reads remain separate. Existing mutation generations reject older fills. Warming never updates browser text, drafts or acknowledged versions, never retries writes, and never warms index/calendar snapshots. Failures leave the mutation successful and may leave content cold. Cache capacity/TTL and process-boundary limitations remain unchanged. This moves an optional Drive read ahead of navigation, adding upstream work even if the user never revisits; immediate navigation may still wait for the in-flight read. No periodic refresh is introduced.
