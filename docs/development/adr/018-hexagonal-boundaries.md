# ADR 018: Explicit hexagonal packages and connection application services

Status: accepted

Date: 2026-10-11

## Context

The journal already owned its storage/clock ports and business rules, but generic
login, disconnect and settings orchestration remained in HTTP delivery. Session
models and lifetime policy shared a package with encryption. The user requested
explicit hexagonal directory names and approved a feature-oriented core, local
cache grouping and the ByteCache name.

## Decision

Use `internal/core/journal` and `internal/core/connection` for neutral models,
application services and consumer-owned ports. Keep related models/rules/services
in feature packages rather than adding global entity/service layers.

Driving HTTP code lives in `internal/driver/web`, including generated API code,
cookie metadata/adaptation, localization and explicit renderer construction.
Driven providers live in `internal/driven/provider/{local,google}`, codecs in
`internal/driven/session` and the system clock in `internal/driven/clock`.
Startup parsing remains in `internal/config`; only `cmd/app` assembles concrete
adapters, services and delivery through Google Wire.

Connection workflows consume a Catalog of lifecycle Gateways without requiring
journal Store construction. The composition root uses the Provider bundle for
registration and account-scoped stores. Connection services own pending-login
validity/state/provider binding, consent/reconnect eligibility, issuance,
revocation and settings folder lookup. HTTP retains malformed/duplicate query
checks, bounded cookie/code parsing, encryption, redirects and error mapping.
Provider-specific OAuth claims, refresh and Drive mechanics stay in Google.
Logout remains a transport-only operation. Successful revocation precedes cookie
removal; failed revocation retains the session and browser drafts.

Preserve existing encrypted JSON fields, key derivation, authenticated purpose,
cookie values/attributes and lifetimes. Synthetic pre-migration ciphertext
fixtures pin compatibility. Stateless replay limits remain as recorded in ADR 002.

The journal Store decorator lives in `internal/driven/cache/journal` (Go package
cachestore), owning its supporting ByteCache port. Concrete in-process storage is
explicitly grouped in `internal/driven/cache/local/{bigcache,lru}`. BigCache remains
selected; LRU is optional. No runtime selector, remote adapter or distributed
coordination is introduced. ADR 017's expiry/invalidation/memory guarantees remain.

Core/journal owns a pure date eligibility rule shared with editor routes. Local
origin validation belongs to neutral connection policy and is shared by startup
validation and the development provider without adapter-to-config imports.
Filesystem operational failures map to ErrUnavailable while preserving error
identity. Metadata listing failures, including disappearance after enumeration,
fail the complete listing rather than silently omitting entries. Missing listing
paths remain empty; malformed unrelated files remain ignored.

Enforce recursive boundaries in `.golangci.yml` using depguard allowlists and deny
rules. Core cannot import transport, adapters or startup config; journal cannot
import connection. Driver can consume the codec but cannot import concrete
providers/cache/clocks/config. Driven code cannot import driver/config; cache
policy and codecs have narrow consumer-specific allowlists. Startup config cannot
import adapters. Test-only driving/driven integration composition is allowed;
production core cannot import shared test fixtures. Temporary acyclic and nested
probes verify the rules and clean up on success/failure.

## Consequences

Application lifecycle policy is directly testable without HTTP, encryption or
storage adapters. Explicit render/catalog initialization propagates errors through
Wire and preserves cache cleanup. Package moves change Go imports and generator
output paths, not browser/API routes or storage locations.

The refactor preserves process-local mutation/cache coordination and accepted
Drive external-writer/deployment-overlap limitations. It does not establish new
performance, live Google, accessibility or release acceptance evidence.
