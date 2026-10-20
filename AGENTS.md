# Agent guide for myjournal-app

## Start here

Read the user's current instructions and inspect affected code before relying on documented behavior or historical passing checks. All required context is tracked; private planning files and chat exports are not prerequisites.

- [Requirements](docs/development/requirements.md): product behavior, recovery invariants and non-goals.
- [Architecture](docs/development/architecture.md): boundaries, current decisions, caching/security guarantees and provenance.
- [Testing](docs/development/testing.md): commands and manual regression procedures.
- [Operations](docs/development/operations.md): configuration, Google, domains and release/rollback procedures.
- [Status](docs/development/status.md): outstanding work, accepted evidence and limitations.
- [OpenAPI](docs/api/openapi.yml): API contract; [SECURITY](SECURITY.md): private vulnerability reporting.

## Setup and checks

Preserve module `myjournal-app`, Go 1.27.2 and existing dependencies. Prefer the standard library. Follow [README prerequisites](README.md#prerequisites): Go, Git, Make and OpenSSL; checks need Node.js 24 or macOS JavaScriptCore, Wire v0.7.0, oapi-codegen v2.8.0, Mockery v3.8.1 and golangci-lint v2.14.0. Do not silently upgrade generators or substitute Mockery v2. Python, Docker and browser QA tools are task-specific.

Fresh clone: `go mod download`, create `.env` from `.env.example` without overwriting an existing file, set SESSION_KEY using `openssl rand -base64 32`, then `make generate`, `make fmt lint test`, separately `make test-architecture`, and `make run`. Never print `.env`. Local defaults require no Google credentials. `make run` loads `.env`; direct go run does not. Keep listener port and base URL aligned; never kill a listener without establishing ownership.

```sh
make fmt lint test       # implementation changes
make test-js            # browser regressions only
make test-architecture  # run separately: temporary forbidden-import probes
python3 -B -m unittest discover -s scripts/python -p 'test_*.py'
make generate-openapi   # after schema edits
make generate-wire      # after Wire edits
make generate-mocks     # after mocked-interface/config edits
```

Edit `cmd/app/wire.go` and regenerate/version wire_gen.go; edit `docs/api/openapi.yml` and regenerate internal/driver/web/api/openapi.gen.go. Preserve declared OpenAPI 3.2.1 compatibility. Mockery output is also generator-owned. Inspect generated diffs; never hand-edit generated files or restore obsolete generated directories. Preserve JavaScriptCore's success-marker check for asynchronous assertions.

Documentation-only changes need path/link checks, consistency review and `git diff --check`, not runtime tests. Report exact fresh commands/results and unavailable tools; historical acceptance is not a fresh run.

## Implementation rules

- Dependencies point inward; core has no transport, rendering, filesystem or provider SDK dependencies. Only cmd/app composes concrete production adapters/codec. Keep HTTP handlers thin and use consumer-owned ports; do not introduce interfaces for every concrete type. Preserve Depguard boundaries and shared Store contracts.
- Validate configuration at startup and pass explicit dependencies. Avoid hidden environment reads, constructors that guess argument meanings, magic policy strings/numbers, speculative flexibility and unrelated refactors. Copy retained mutable inputs when needed.
- Validate domain rules before storage/external calls. Preserve opaque versions, serialization, existing data and wrapped error identity; use errors.Is/errors.As. Distinguish absence from failure and never suppress errors to fabricate success.
- Propagate context through I/O and cancelable lock waits. Bound memory, coordination, concurrency and external work. Prefer operation-scoped reuse before persistent caching. Preserve invalidation/generation guards and state where guarantees stop at process boundaries.
- Use focused functions, guard clauses and useful exported documentation. Place compile-time interface assertions immediately below the implementing struct, grouped for that struct.
- `make fmt` owns handwritten Go formatting. Imports group standard library, third-party, then myjournal-app, alphabetically within each group including aliases. Generated code stays excluded from formatter rewrites.

## Safety constraints

Read the relevant guide before changing these behaviors:

- Production is a stateless Google Drive bridge, with authenticated encrypted cookies and bounded process memory; local filesystem storage is development-only. Never add a production database/session store or broaden OAuth scopes as a workaround.
- Preserve plain-text UTF-8/LF files, appProperties discovery, read-without-create, non-destructive duplicate handling, confirmed trash and future-date rejection. Drive compare/upload is non-atomic; do not claim external/multi-instance safety.
- Preserve text and drafts on failures, conflicts, expiry and ambiguous acknowledgements. Keep account/date/tab scope, fresh reconciliation, one in-flight save, acknowledged-version chaining and explicit review of recovered drafts. Authentication recovery never silently navigates away, switches accounts or treats expiry as successful logout.
- Logout/revocation acknowledgement precedes draft cleanup. Cleanup-only retries never repeat revocation. Browser storage failures must remain truthful. Closed-tab backup expiry is accepted.
- Keep fixed 30-day session age, ten-minute pending-login state/PKCE/nonce, same-account omitted-refresh preservation and startup HTTPS/Secure-cookie checks. Testing Google refresh tokens may expire sooner.
- Preserve same-origin protections/CSP, safe text rendering, external local scripts, host-only configured-origin HSTS and query-free logs. Never log/commit credentials, tokens, codes, callback queries, cookies, account claims or journal content. Never request decrypted Railway variables.
- Cache freshness, singleflight bounds and warming remain optional performance policy, not atomicity guarantees. Never seed snapshots from potentially out-of-order save acknowledgements. Drain HTTP and join warming before cache cleanup.
- Preserve EN/ID flat locale keys, accessible theme-aware assets, keyboard/live-region behavior and latest-request-only calendar rendering. Keep theme controls in the header, Monday week start and current product non-goals.

## Google and releases

Restricted Google Testing release acceptance is complete; do not reopen accepted experiments routinely or describe it as unrestricted Console publication. Consult status for evidence/limits. Console publishing and live release actions require task authorization.

The intended release policy is published non-prerelease GitHub Releases only, with main ancestry/code checks, serialized deployments, clean source archives, exact deployment-ID polling and public health verification. The current manual-dispatch discrepancy is documented in status. PR/merge/tag-push paths must remain offline. Do not deploy to validate a documentation change.

Railway IaC lives in `.railway/railway.go` in its own module; see [.railway/README.md](.railway/README.md). Keep variables unmanaged, the pinned-plan restrictions, existing production target, one Singapore replica and post-apply convergence checks. Do not restore railway.json or attach a GitHub branch source. Preserve SESSION_KEY across rollback; rollback does not revert Drive writes. RAILWAY_TOKEN stays a secret, never a plain variable.

## Tests

Use Testify assert for independent checks and require for prerequisites. Never call fatal assertions from worker goroutines; return results to the test goroutine. Prefer focused assertions with explicit expected values. Use named table-driven subtests for related cases sharing setup/assertion shape; keep stateful/concurrency fixtures explicit.

Use Mockery-generated interface mocks, not handwritten stubs or nil-interface embeddings. Configure `.mockery.yml`, regenerate with pinned v3 and version output. Create mocks with each subtest's t; prefer typed EXPECT and constrain meaningful arguments/counts without irrelevant ordering. Keep generated mocks test-only. Shared fixtures must represent genuinely shared setup and stay independent of production code.

Add meaningful regression coverage for changed observable behavior, boundaries, cancellation and concurrency. Use fixed clocks, temporary directories and mocked external services. Do not introduce tests that merely mirror low-impact reversible edits.

## Working discipline and documentation

Continue authorized reversible work autonomously. Inspect git status and preserve staged/user edits. Do not delegate unless explicitly requested by the user or applicable instructions. Do not commit, push, publish, merge or resolve remote threads without task authorization.

Commit/PR subject format: `<type>: [<scope>] <subject>` when useful, otherwise `<type>: <subject>`, concise present tense. Types: feat, fix, docs, style, refactor, test, chore. Branch convention: `<type>/<short-kebab-case-subject>`. Use Squash and merge for authorized merges unless the user requests otherwise, with a semantic squash subject. PR descriptions must follow [the PR template](.github/pull_request_template.md): a title heading, Description, Type of Change checkboxes, Changes Made and Validations. Fill in the applicable change types and provide concrete changes, validation evidence and material limits; remove placeholder comments and empty bullets.

Keep README focused on durable setup/product usage and retain make generate in quick start. Record each fact once: requirements own product contracts; architecture owns consequential decisions; testing owns procedures; operations owns runbooks; status owns outstanding work and concise evidence/limitations. Update links instead of copying prose. Do not recreate phase diaries, per-decision files, dated reviews or completed implementation plans. Use Git history for obsolete records. Distinguish requirements, automated checks, human reports and accepted waivers; a backlog is not authorization to implement it.
