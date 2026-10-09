# myjournal-app

A Go-based journal application scaffolded to follow the implementation guide in `agents/guide.md`.

## Current status

The local development flow is working end-to-end:

- app shell and router are in place
- entry pages and API routes are versioned under `/api/v1`
- future dates are rejected
- the branded 404 page is connected to the app
- save status transitions are visible and styled
- previews use the first meaningful line after trimming leading blank space
- the local provider is used for day-to-day development while the Google provider remains pending human-assisted setup

## Quick start

Create `.env` from `.env.example` and set `SESSION_KEY` to the output of `openssl rand -base64 32`.

```bash
make generate
make run
```

Then visit <http://localhost:3030/healthz>.

## Development

```bash
make fmt
make lint
make test
```

`make test` runs the editor interaction regressions and `go test ./...`. The JavaScript checks use Node.js when available, or the built-in JavaScriptCore shell on macOS; no npm packages are required. Run only those checks with `make test-js`. Manual browser acceptance steps are in `docs/development/editor-recovery-checks.md`.

## Architecture

The project is organized into an application shell, core domain packages, and adapters following the guide's lightweight hexagonal structure.

## Recent verified behaviors

- entry routes render a branded 404 instead of the raw Go default text
- entry preview intentionally ignores leading blank lines and starts from the first real line of user-written text
- save status wording is intentionally short: `Saved`
- editor state resets to neutral when content becomes dirty again after a successful save
