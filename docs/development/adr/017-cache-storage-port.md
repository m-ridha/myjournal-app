# ADR 017: Separate journal cache policy from cache storage

Status: accepted

Date: 2026-10-11

## Context

`internal/cachestore` previously combined journal-specific snapshot rules with a
custom map/list LRU. The user requested BigCache and an adapter boundary that
makes changing cache implementations easier while preserving journal behavior.

## Decision

`cachestore.ByteCache` is a consumer-owned, context-aware byte-storage port with
Get, Set, Delete and DeletePrefix. It distinguishes misses and capacity rejection
from failures and requires caller-owned bytes. `internal/driven/cache/local/bigcache`
implements the port with the installed `github.com/allegro/bigcache` v1.2.1.
Only `cmd/app` constructs the production adapter; Wire supplies cleanup after
request draining. No Redis adapter is introduced. By follow-up user decision, the former map/list
LRU is retained in `internal/driven/cache/local/lru` as an optional backup adapter;
BigCache remains the only production selection. The LRU implements the same port
with a 2048-entry default, explicit capacity override, lazy TTL expiry, copied
bytes and cancelable lock waits. It owns no goroutines and needs no cleanup.
Its entry-count bound is not a byte-memory limit. Selecting it requires an
explicit composition-root change; no runtime fallback or environment selector
is added. The journal decorator lives in `internal/driven/cache/journal`; providers live
under `internal/driven/provider`. ADR 018 records the package migration.

The decorator retains account isolation, serialized snapshots, absolute expiry,
Get bypass, preview-budget keys, operation forwarding and the scalar generation
guard. Mutation invalidation follows ADR 015: remove every preview budget for the
affected date, independently retain proven year/all-years membership after a
successful update, preserve unrelated expiry, and never seed from acknowledgements.
Account namespaces use unambiguous base64url encoding; this is not encryption.

BigCache owns eviction and reclamation. Configure 16 shards, a 16 MiB byte-queue
limit, 2048 expected entries, a 512-byte initial size estimate, one-second cleanup,
and no verbose logging. Expected entry count is an allocation hint, not a cap.
Queue size excludes Go map and runtime overhead and is not a process memory
ceiling. Entries that exceed one shard's capacity or the library's key-length
representation are rejected before insertion; backing results still succeed.
The application TTL remains two minutes. Snapshots enforce precise expiry because
BigCache reads can return entries awaiting cleanup.

Cache failures cannot change a successful journal mutation into an error.
Unexpected backend or decoding errors disable caching for that shared coordinator
until restart. Capacity rejection skips a fill without disabling the cache.
Cancelled or failed invalidation also disables reuse, including when cancellation
prevents acquisition of the coordination lock. Channel gates make cache lock waits
cancelable. Backing journal I/O runs outside those gates.

## Consequences

The old 2048-entry LRU guarantee is replaced by bounded BigCache byte queues and
its eviction policy. There is no performance improvement claim. Prefix deletion
scans the bounded backend and excludes adapter writes; no second unbounded key
inventory is maintained. Oversized indexes are read from backing storage normally.

Shared cache ownership and generation coordination remain process-local. A Redis
implementation of the byte port alone is insufficient for multi-instance safety;
distributed invalidation/fill coordination requires a separate design. Existing
external-writer and deployment-overlap limitations remain unchanged.

Tests cover real-adapter storage/eviction/cancellation/lifecycle and the cached
journal store contract, plus decorator expiry, races, isolation, mutation failures,
serialization and backend-failure fallback. Depguard and temporary probes enforce
the new import boundaries without introducing infrastructure into journal or web.
