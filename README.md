# myjournal-app

<p align="center">
  <img src="assets/static/logo/myjournal-app-logo-120x120.png" alt="myjournal-app Logo" width="120" height="120">
</p>

A Go app for one plain-text journal entry per day, stored in the user's Google Drive. The app uses a JSON API and vanilla JavaScript with server-rendered templates; production has no user database or persistent journal storage.

Contributor instructions are in [AGENTS.md](AGENTS.md). See [requirements](docs/development/requirements.md), [architecture](docs/development/architecture.md), [testing](docs/development/testing.md), [operations](docs/development/operations.md) and [current status](docs/development/status.md).

## Features

- One UTF-8 plain-text entry per day, with browser-time-zone dates and editable past entries.
- Manual save and idle autosave, per-tab draft recovery, conflict-resolution controls and confirmed trash.
- Previews for your latest 10 entries, calendar navigation with a month/year picker for older notes, light/dark themes and account settings.
- Google sign-in and Drive storage, with a local provider for development.
- English and Bahasa Indonesia interfaces, selectable from the header or Settings, including before sign-in. Your choice stays in this browser after sign-out; English is the default.

The app is online-only. Browser drafts provide recovery rather than offline synchronization. Checksum comparisons are non-atomic; simultaneous external Drive edits are unsupported. See [deployment limitations](docs/development/status.md#accepted-technical-limits) for deployment and writer constraints.

## Prerequisites

Install Go **1.27.2** (the version declared in `go.mod`), Git, Make, and OpenSSL. Development checks also need **Node.js 24**; macOS JavaScriptCore is a supported alternative for the JavaScript tests. No npm install, database, or Google credentials are required for local-provider development.

Install the Go development tools at the repository's pinned versions:

```sh
go install github.com/google/wire/cmd/wire@v0.7.0
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0
go install github.com/vektra/mockery/v3@v3.8.1
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
```

Put the installation directory on your shell's PATH. Go uses `GOBIN` when configured, otherwise `$(go env GOPATH)/bin`:

```sh
export PATH="${PATH}:$(go env GOPATH)/bin"
# If you configured GOBIN, add that directory instead.
```

Persist the PATH setting in your shell configuration if needed. Verify the tools are available:

```sh
go version
wire help
oapi-codegen -version
mockery version
golangci-lint version
```

The `.mockery.yml` configuration requires **Mockery v3**; `go install github.com/vektra/mockery/v2@latest` installs a different major version. Prefer the pinned commands above over `@latest` so generated code is reproducible. Testify and other application/test libraries are downloaded through Go modules, without separate CLI installation.

## Quick start

After cloning the repository and entering its root directory:

```sh
go mod download
test -e .env || cp .env.example .env
openssl rand -base64 32
```

Set `SESSION_KEY` in `.env` to the generated value. Keep `.env` private. Defaults use the development-only local provider at port 3030 with files under `.devdata`; Google credentials can stay empty.

```sh
make generate
make fmt lint test
make test-architecture
make run
```

Run `make test-architecture` separately from other checks because it temporarily adds forbidden imports. `make run` stays running until you stop it; use Ctrl+C to stop your development server. Alternatively, after setting up `.env`, `make run` alone starts the app using the committed generated code.

Open <http://localhost:3030/> and use the local sign-in. Health is available at <http://localhost:3030/healthz>. `make run` loads `.env`; `go run ./cmd/app` does not. Keep `PORT` and `APP_BASE_URL` aligned. Local-provider and Google test-mode servers bind only to loopback; production Google deployments listen on all interfaces.

`make generate` runs OpenAPI, Wire and Mockery generation. Generated files are committed; inspect generated diffs before committing. Run `make generate-mocks` after changing mocked interfaces or `.mockery.yml`.

## Build version

Local runs omit the footer version. The published-release workflow writes its validated GitHub release tag into `RELEASE_VERSION` inside the deployment archive. Docker embeds that value into the Go binary using `-ldflags`, and displays it as plain text. No runtime GitHub request is needed. `RELEASE_VERSION` is not stored in the repository and is optional for Docker builds; Docker leaves the version empty when it is absent. Accepted tags include `v0.3.0-staging` and `v0.5.1-closed-beta`; the three core numbers cannot have leading zeros, while suffix identifiers may (for example, `beta.01`). `+` build metadata is not supported. Nonempty embedded versions also appear as a `version` field in application logs.

## Development

```bash
make fmt lint test
make test-architecture
```

Run architecture probes separately from other lint/build processes because they temporarily add forbidden imports. Lint requires golangci-lint v2.14.0. `make test` runs seven JavaScript regression suites and `go test ./...`; JavaScript uses Node.js or macOS JavaScriptCore without npm packages. Run only browser interaction regressions with `make test-js`. Go unit tests use Testify assertions and Mockery-generated interface mocks. Prefer named table-driven subtests for cases sharing the same setup and assertions; keep stateful storage/HTTP fixtures and concurrency scenarios explicit. Generated mocks are test-only and checked for drift in PR CI.

Optional tooling is separate from the local quick start: Python 3 runs the helper-script tests (`python3 -B -m unittest discover -s scripts/python -p 'test_*.py'`); Docker with a running daemon is needed for `docker build -t myjournal-local .`. The [testing guide](docs/development/testing.md) documents the additional browser/accessibility setup. None of these tools is needed by `make run`.

The testing guide also covers editor recovery and manual UI checks. PR creation and updates run formatting, lint, tests and architecture checks.

## Architecture

`internal/core/journal` owns domain rules and storage/clock ports. Local and Google providers implement adapters; the web layer handles JSON APIs under `/api/v1` and browser pages. `cmd/app` composes dependencies with Wire. Embedded assets use `html/template`, local CSS, scripts and SVGs. See AGENTS.md for boundaries and durable conventions.

## Google integration testing

See [Google integration checks](docs/development/testing.md#google-integration) for the localhost opt-in, Console callback and synthetic-data checks. Keep `GOOGLE_TEST_MODE=false` for ordinary local development. `GOOGLE_TEST_MODE=true` permits integrated Google testing only on localhost/loopback; non-test Google requires an HTTPS origin and Secure cookies. Preserve the approved `openid`, `email`, `profile` and `drive.file` scopes.

## Railway releases

[Operations](docs/development/operations.md) documents configuration, privacy and rollback. Publishing a non-prerelease GitHub Release with a `vX.Y.Z` tag and optional version suffix deploys that tag's source after main-ancestry and code checks. A tag push alone does not deploy. The workflow polls the exact Railway deployment ID and verifies public health.

Production infrastructure is defined in [`.railway/railway.go`](.railway/railway.go), with its own pinned Go SDK module. The same published-release job validates and applies a pinned infrastructure plan before uploading release source; PRs, merges and tag pushes do not apply infrastructure or deploy. Railway variables and secrets remain managed on Railway. See [IaC operations](.railway/README.md).

The workflow retains the repository-variable gate `RAILWAY_RELEASE_READY=true`. Configure `RAILWAY_PROJECT_ID` as a repository variable, `APP_BASE_URL`, `RAILWAY_SERVICE_ID` and `RAILWAY_ENVIRONMENT_ID` as production-environment variables, and `RAILWAY_TOKEN` as a secret. Keep one replica and branch autodeploys disabled. Maintain the same `SESSION_KEY` across restart/rollback; rollback does not revert Drive writes. Never commit credentials, cookies or journal content.
