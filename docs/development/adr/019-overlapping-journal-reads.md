# ADR 019: Share bounded overlapping journal reads

Status: accepted

Date: 2026-10-12

## Context

Operation-scoped discovery and the journal TTL cache reduce repeated browsing, but simultaneous cold requests still duplicate storage work. The user approved request collapsing, then explicitly selected `golang.org/x/sync/singleflight` and requested a provider-independent boundary.

## Decision

The journal cache decorator owns shared in-flight coordination around the existing `journal.Store` port. It collapses index misses, editor reads, identical preview batches and matching preview dates across different batches. All providers wired through this decorator benefit. The byte-cache storage adapters remain unaware of journal operations or concurrency policy. No additional core interface is required.

Use `singleflight.DoChan` for duplicate suppression and result publication. A wrapper bounds registrations, propagates request cancellation and retries a cancelled leader through a surviving caller's own Store/context. Keys include provider/account namespace, generation and read parameters. Different devices/grants for the same account share successful reads. A follower of an authentication failure retries once directly through its own Store while retaining its admission slot; a failing leader returns its own error. Preview keys include the byte budget; batch keys canonicalize dates. The composition root validates the connection through the provider before wrapping the Store; no credentials or credential hashes enter coordination keys. Persistent indexes/previews retain their existing account-scoped keys and TTL policy.

Completed flight results and errors are discarded. Editor content never enters the persistent byte cache; sequential editor reads still resolve and download fresh content. Cache fills happen before flight publication and retain existing generation guards. Returned slices/maps are copied. No credential or journal text is logged.

Bound each coordinator to 64 admitted distinct keys and 320 caller registrations, including cancelled callers whose result channels have not yet drained. Separate batch and individual-read coordinators avoid nested admission deadlocks; together they admit at most 128 distinct keys and 640 registrations. Each leader has a request-derived 30-second deadline. Cancelled callers return promptly; a bounded drainer releases admission when the underlying result arrives. Context-aware adapters must stop their I/O on cancellation. Overload returns the existing unavailable error. Cache-decorated preview batches use at most eight workers. The Google adapter retains its four-worker direct-batch limit and starts only as many workers as requested dates; decorated singleton calls therefore permit eight downloads, not thirty-two.

Save and trash attempts advance a fixed, account-striped generation before and after the attempt. New reads cannot join a flight from before a completed attempt. Mutations themselves are never collapsed or retried by this decorator. Old readers may still receive an earlier valid snapshot; this is not linearizable storage or atomic external-writer protection. Credential/grant changes do not separate flights; providers must validate that credentials belong to the stated account. Disconnect does not actively cancel already-admitted reads.

Google retains bounded operation-scoped discovery reuse (ADR 015), rather than provider-wide read coordination. Its local discovery uses `singleflight.Do` with short metadata locks, allowing the decorator's parallel preview calls to reuse discovery without holding a mutex over network I/O. Discovery generations prevent post-clear fills/joins from reusing an earlier operation snapshot. Pre-upload metadata checks stay independent.

## Consequences

The shared decorator reduces duplicate public Store work for every provider. Unlike the superseded Google-specific coordinator, it cannot merge individual folder queries across different Store operations. Controlled cold Entries requests therefore use more metadata calls than that narrower implementation, while preserving the download reduction. Batch-level coordination avoids scattering identical preview work across many independent operation views.

Sharing remains process-local. No Redis dependency, longer TTL, Google scope change, background writer or server-side session store is introduced. Existing TTL staleness, non-atomic Drive compare/upload and deployment-overlap risks remain. Synthetic request counts establish reduced work, not live latency or a passed performance target. See [measurements](../performance-benchmark.md#overlapping-drive-reads-2026-10-12).
