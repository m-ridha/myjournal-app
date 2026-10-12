# ADR 020: Two-minute entry content snapshots

Date: 2026-10-13
Status: Accepted by user

## Decision

Cache ordinary full-entry reads through the shared journal decorator using the existing two-minute TTL and bounded byte storage. Retain text, existence and opaque version together. Cache confirmed ErrNotFound as an explicit absence snapshot with the same TTL and invalidation. Other errors are not cached. Provider/account/date isolate snapshots. This supersedes ADR 019's uncached sequential editor-read policy.

PUT and DELETE invalidate the affected entry snapshot alongside its preview; unrelated snapshots retain their original expiry. Preserve existing conservative invalidation after failed mutation attempts, because acknowledgements can be ambiguous. Generation guards reject older in-flight fills. Never seed snapshots from save acknowledgements.

Recovery needs actual storage state: core Service.OpenFresh uses the optional consumer-owned FreshEntryStore port, falling back to ordinary Get for uncached providers. The decorator's GetFresh bypasses snapshot reads/writes and uses a separate bounded flight key. HTTP maps Cache-Control: no-store to OpenFresh; responses prohibit browser/intermediary caching. Browser draft recovery, failed-save reconciliation and conflict actions explicitly request fresh reads. Ordinary opens may reuse snapshots.

## Consequences

Warm opens avoid Drive discovery/download; cold opens still require them. Direct Drive edits and writes handled by another process can remain invisible for up to two minutes. Invalidation coordinates only the current process. Existing non-atomic external-writer and deployment-overlap limitations remain. Full journal content now resides in bounded process memory until logical expiry/eviction; lazy cache expiry does not guarantee immediate physical byte erasure. Privacy copy reflects the expanded cache content. No new persistence, scope, Redis dependency, TTL increase or offline mode is introduced.
