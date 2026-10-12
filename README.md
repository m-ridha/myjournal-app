# myjournal-app

A Go app for one plain-text journal entry per day, stored in the user's Google Drive. The app uses a JSON API and vanilla JavaScript with server-rendered templates; production has no user database or persistent journal storage.

Contributor instructions are in [AGENTS.md](AGENTS.md). See [progress](docs/development/PROGRESS.md), [findings](docs/development/findings.md), and the [phase acceptance requirements](docs/development/phase-acceptance.md) for evidence and remaining work.

## Features

- One UTF-8 plain-text entry per day, with browser-time-zone dates and editable past entries.
- Manual save and idle autosave, per-tab draft recovery, conflict-resolution controls and confirmed trash.
- Paginated previews, calendar navigation, light/dark themes and account settings.
- Google sign-in and Drive storage, with a local provider for development.
- English and Bahasa Indonesia interfaces, selectable in Settings. Your choice stays in this browser after sign-out; English is the default.

The app is online-only. Browser drafts provide recovery rather than offline synchronization. Checksum comparisons are non-atomic; simultaneous external Drive edits are unsupported. See [ADR 016](docs/development/adr/016-restricted-google-release.md) for deployment and writer constraints.

## Quick start

Use Go 1.27.2. Copy `.env.example` to `.env` and set `SESSION_KEY` to the output of `openssl rand -base64 32`. Keep `.env` private. Defaults use the development-only local provider at port 3030 with files under `.devdata`.

```bash
make generate
make run
```

Open <http://localhost:3030/> and use the local sign-in. Health is available at <http://localhost:3030/healthz>. `make run` loads `.env`; `go run ./cmd/app` does not. Keep `PORT` and `APP_BASE_URL` aligned. Local-provider and Google test-mode servers bind only to loopback; production Google deployments listen on all interfaces.

Install Wire and oapi-codegen before running `make generate`; see AGENTS.md for the required versions and commands. Generated files are committed, and the quick start regenerates them from their source definitions. Inspect generated diffs before committing.

## Development

```bash
make fmt lint test
make test-architecture
```

Run architecture probes separately from other lint/build processes because they temporarily add forbidden imports. Lint requires golangci-lint v2.14.0. `make test` runs six JavaScript regression suites and `go test ./...`; JavaScript uses Node.js or macOS JavaScriptCore without npm packages. Run only browser interaction regressions with `make test-js`.

The [editor recovery checklist](docs/development/editor-recovery-checks.md) and [UI acceptance checks](docs/development/ui-acceptance-checks.md) document manual coverage. Optional accessibility tooling is separate from ordinary CI. PR creation and updates run formatting, lint, tests and architecture checks.

## Architecture

`internal/core/journal` owns domain rules and storage/clock ports. Local and Google providers implement adapters; the web layer handles JSON APIs under `/api/v1` and browser pages. `cmd/app` composes dependencies with Wire. Embedded assets use `html/template`, local CSS, scripts and SVGs. See AGENTS.md for boundaries and durable conventions.

## Google integration testing

See [Google live checks](docs/development/google-live-checks.md) for the localhost opt-in, Console callback and synthetic-data checks. Keep `GOOGLE_TEST_MODE=false` for ordinary local development. `GOOGLE_TEST_MODE=true` permits integrated Google testing only on localhost/loopback; non-test Google requires an HTTPS origin and Secure cookies. Preserve the approved `openid`, `email`, `profile` and `drive.file` scopes.

## Railway releases

[Railway preparation](docs/development/railway-preparation.md) documents configuration, privacy and rollback. Publishing a non-prerelease GitHub Release with a stable `vX.Y.Z` tag deploys that tag's source after main-ancestry and code checks. A tag push alone does not deploy. The workflow polls the exact Railway deployment ID and verifies public health.

The workflow retains the repository-variable gate `RAILWAY_RELEASE_READY=true`. Configure `RAILWAY_PROJECT_ID` as a repository variable, `RAILWAY_SERVICE_ID` and `RAILWAY_ENVIRONMENT_ID` as production-environment variables, and `RAILWAY_TOKEN` as a secret. Keep one replica and branch autodeploys disabled. Maintain the same `SESSION_KEY` across restart/rollback; rollback does not revert Drive writes. Never commit credentials, cookies or journal content.
