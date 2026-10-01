# Findings Log

## Bootstrap

- `go.mod` was already present and has been preserved for this repository.
- A minimal Go bootstrap was created to satisfy the Phase 0 acceptance criteria.
- `make fmt lint test` passed successfully in the current workspace after installing the correct golangci-lint v2 toolchain and aligning the config to the guide’s architecture.
- Google OAuth environment values are now present in the local project environment for the next Phase 1 work.

## Verified local app behavior

- The app routes are versioned under `/api/v1` and the entry pages use the expected date-based paths.
- Future dates are rejected by both the router and the API layer.
- A missing or invalid route renders the branded not-found page instead of the raw default Go `404 page not found` response.
- The entry preview logic now trims leading blank space and uses the first meaningful text line as the preview, without counting the entire entry length as the preview length.
- The editor shows a visible save-state cycle: saving → saved → neutral while dirty again.
- The UI text was shortened from `Saved locally` to `Saved` to keep the status cleaner and less noisy.
- Client-side draft persistence keeps text locally when a save fails or the page is left mid-edit.

## Phase 1: Google OAuth spike

- The provider seam and Google config scaffolding are in place.
- A throwaway PKCE spike is available in `spike/s01_google_oauth` and is designed to exercise the live login and initial Drive API read path.
- Live sign-in, consent prompts, refresh-token validation, and Drive folder behavior remain for the human-assisted Google Cloud / browser flow.

## Unverified assumptions

- Google Drive folder metadata behavior will be confirmed in Phase 1 spike experiments.
- Theme and browser-specific behaviors remain to be validated in the UI phase.
- Real Google Drive provider reliability and refresh-token handling remain pending human-assisted verification.

## API contract note

- Recommended future practice: define the HTTP API contract with OpenAPI Schema (OpenAPI 3.2.1) once the app reaches the API-routing phase. This keeps route definitions, payload shapes, and status codes explicit and reviewable without changing the current phased implementation plan in the guide.
- The generated OpenAPI server and contract are already in place under `docs/api/openapi.yml` and `internal/web/api/`.

## Verification status

- `go test ./...` passed in the current workspace after the latest UI and preview updates.

## Restructuring verification

- Generated OpenAPI code now lives at `internal/web/api/openapi.gen.go`; the contract source is `docs/api/openapi.yml`.
- `journal.Store` and `journal.Clock` are separate ports in `internal/journal/store.go`.
- Web entry handlers use an injected `journal.Service`; the service owns entry listing, calendar construction, opening, saving, and deletion.
- The local provider passes the shared `internal/journal/storetest` contract suite, including stale-save conflict behavior after deletion.
- `google/wire` was regenerated with `go generate ./cmd/app` after the clock dependency moved to the composition root.
- The expanded depguard configuration passes `golangci-lint run` with 0 issues. Deliberate violation canaries remain a follow-up if the lint configuration needs stronger automated proof.
- Middleware is intentionally only scaffolded at `internal/web/middleware`; implementation is deferred until the relevant feature phases.
- The in-memory cache is created once in `cmd/app` and shared by account-keyed `journal.Store` decorators. It uses a two-minute TTL and a 2048-entry LRU bound.
- Cache tests cover TTL expiry, capacity eviction, account isolation/shared reuse, preview-size keying, and mutation invalidation.
- `journal.Service.Page` reads newest year indexes only as far as needed to fill a page and detect `HasNext`, then fetches previews for that page only.
- `journal.Service.Month` uses one year index for entry markers rather than issuing one `Get` per calendar day.

## Localization

- `assets/locales/en.json` is the canonical source for user-facing UI copy.
- The locale source uses flat, dotted string keys such as `landing.feature.quiet.title`; `internal/i18n` loads it directly as a string map.
- The same flat catalog is embedded for browser scripts, so browser and Go template lookups use identical keys without load-time flattening.
- `internal/i18n` loads the embedded catalog once and provides message lookup plus the locale's date layout.
- Go templates call the `t` template function. The base layout embeds the JSON catalog in an escaped `data-messages` attribute, which `app.js` and `theme.js` read for status text, calendar labels, list states, and theme-toggle accessibility text.
- API handler errors and generated OpenAPI parameter errors use catalog messages instead of exposing raw Go/provider error text.
- Missing keys fall back to the key itself to make incomplete translations visible during development.
- Account email, journal dates, and user-written entry previews are runtime data rather than translatable interface copy; their surrounding labels are catalog messages.
- New user-visible template, JavaScript, or handler copy should add a key to `assets/locales/en.json` and use the key at the point where it is rendered.

## Authentication boundary and startup validation

- Entry APIs require a decryptable session cookie with provider/account identity and an enabled provider; missing or invalid sessions receive HTTP 401.
- Editor, entries, calendar, and settings pages redirect to the landing page without a valid session.
- The request service factory now resolves the store from the session provider and connection, and scopes the cache key by provider/account.
- Configuration parsing rejects invalid ports and `COOKIE_SECURE` values. Validation requires an absolute HTTP(S) base URL, a base64 session key decoding to 32 bytes, a supported log level, and known provider names.
- The local provider can only be registered for exact `localhost` or loopback IP hostnames; `localhost.evil.com` is rejected.
- Startup refuses the Google provider while OAuth and Drive storage remain stubs, instead of allowing placeholder credentials to appear successful.
- Session expiration and key rotation remain open; `IssuedAt` is required but no maximum session age is enforced yet.
