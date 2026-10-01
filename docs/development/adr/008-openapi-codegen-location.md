# ADR 008: Keep OpenAPI Source at the Repository Boundary

Keep `docs/api/openapi.yml` as the source of truth for the HTTP contract, while placing generated Go code under `internal/web/api/openapi.gen.go`. The contract is a documentation/API artifact, while the generated implementation is an internal detail of the web adapter and should not be importable by packages outside this module or excluded from architecture checks.
