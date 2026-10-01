# ADR 009: Use Google Wire for Dependency Wiring

Use `google/wire` for dependency injection, with `cmd/app/wire.go` as the injector source and the committed `cmd/app/wire_gen.go` as generated composition-root code. This keeps provider implementations and runtime construction in `cmd/app`, makes the dependency graph reviewable, and avoids a hand-maintained composition function drifting from the declared providers.
