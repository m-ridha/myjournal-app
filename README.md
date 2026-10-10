# myjournal-app

A Go-based plain-text journal app. Contributor instructions are in [AGENTS.md](AGENTS.md); see [progress](docs/development/PROGRESS.md) and the [phase acceptance checklist](docs/development/phase-acceptance.md) for implementation status and remaining requirements.

## Current status

The implemented local development flow includes:

- app shell and router are in place
- date-specific entry pages use `/entries/YYYY-MM-DD`; API routes are versioned under `/api/v{version_number}`
- future dates are rejected
- the branded 404 page is connected to the app
- save status transitions are visible and styled
- previews use the first meaningful line after trimming leading blank space
- the local provider is used for day-to-day development; Google OAuth and Drive support localhost test mode and HTTPS/Secure-cookie acceptance deployments; automated production releases remain gated on live acceptance

## Quick start

Create `.env` from `.env.example` and set `SESSION_KEY` to the output of `openssl rand -base64 32`.

```bash
make generate
make run
```

Then visit <http://localhost:3030/healthz>.

## Development

```bash
make fmt lint test
```

`make test` runs the editor interaction regressions and `go test ./...`. The JavaScript checks use Node.js when available, or the built-in JavaScriptCore shell on macOS; no npm packages are required. Run only those checks with `make test-js`. Manual browser acceptance steps are in `docs/development/editor-recovery-checks.md`.

## Architecture

The project is organized into an application shell, core domain packages, and adapters following the guide's lightweight hexagonal structure.

## Recent verified behaviors

- entry routes render a branded 404 instead of the raw Go default text
- entry preview intentionally ignores leading blank lines and starts from the first real line of user-written text
- save status wording is intentionally short: `Saved`
- editor state resets to neutral when content becomes dirty again after a successful save

## Google integration testing

See [Google live checks](docs/development/google-live-checks.md) for the localhost-only opt-in, Console callback, and integrated acceptance sequence. Keep `GOOGLE_TEST_MODE=false` for normal local development. This switch cannot enable Google on a production hostname.

## Railway releases

[Railway preparation](docs/development/railway-preparation.md) documents configuration, privacy, rollback and remaining acceptance. Published non-prerelease GitHub Releases tagged `vX.Y.Z` deploy their checked-out source using a pinned Railway CLI. Deployment completion and public health are checked. Google requires HTTPS and Secure cookies outside localhost test mode; release automation remains gated until public acceptance passes.
