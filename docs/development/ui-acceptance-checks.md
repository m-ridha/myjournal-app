# Focused UI acceptance checks

Covered on 2026-10-09 in Chrome 155.0.8059.40 with isolated local-provider data:

- With no mj-theme preference, initial device dark/light is applied before CSS and device changes update the page. Space on the theme toggle chooses explicitly; later device changes cannot override it and reload preserves it.
- Keyboard Save accepts long plain text, emoji, accented characters, many lines, and a long unbroken sequence; wrapping does not lose text.
- Editor, Entries, Calendar, Settings, Help, Privacy, and styled 404 render in both themes at 390px and at 200% CSS zoom without horizontal page overflow. Visible empty-text buttons have accessible labels.
- Reduced-motion media preference removes calendar transitions. Editor status retains its live region; the future-day control is disabled.
- No browser page exceptions during those flows.

Reproduce using synthetic text and a disposable local data directory. Automated theme state regressions are in tests/theme.test.js and run via make test-js. Architecture checks run separately via make test-architecture; its temporary probes must not overlap another build/lint process.

Remaining evidence needed before full UI acceptance:

- Native browser 200% zoom, thorough keyboard focus order/visibility across every action and error/conflict state, and screen-reader announcements. CSS zoom is only a reflow probe.
- Full accessibility acceptance beyond the automated text-contrast/naming and keyboard coverage below, including native screen-reader announcements and any untested interaction states.
- Separate externally sourced icon provenance: the trash SVG was user-provided without its original source/license. The Monochrome visual-reference review is recorded in ADR 004, with no copied theme source detected.

- Actual Google folder links, revocation, and authentication remain provider acceptance, not local UI results.

The physical-phone waiver remains in effect; narrow Chrome viewport coverage is not physical-device evidence. Native Safari is not included in this fresh focused run.

Resolved architecture decision: the user explicitly retained JSON API and vanilla JavaScript and excluded HTMX (ADR 014). The HTMX assessment remains historical research, not an open acceptance gate.

## Automated accessibility follow-up (2026-10-10)

Chrome 155.0.8059.40 and axe-core 4.10.3 passed 18 WCAG A/AA-tagged scans: seven routes plus failed-save and conflict states in both themes. The audit initially reproduced a dark Settings Disconnect text contrast failure (2.55:1) and a calendar naming issue. Using the existing danger theme variable and adding role=group to the named calendar container resolved them. All visible enabled links/buttons/editable textareas were reached by Tab and had a computed keyboard focus indicator on the seven routes and conflict editor. This checks reachability/indicators, not the subjective quality of every focus order or state.

The scanner leaves arrow-glyph contrast as incomplete. Supplemental computed foreground/background measurements for enabled previous/next controls passed 4.5:1, with observed ratios 13.98–17.40:1. These measurements do not certify all non-text control boundaries or focus-indicator contrast. Incomplete results remain in the report instead of being discarded. Native screen-reader and native zoom checks remain open.

The optional scripts/check-accessibility.py tool uses an owned ephemeral-port server, temporary synthetic local data, and a random session key. It shuts down the server and removes data. It is separate from ordinary CI and adds no runtime dependency or page request to a CDN. Supply a locally downloaded axe script; do not run this against real journal data.

Example setup/run (requires Python, installed Chrome, and network access for initial QA tools):

```sh
python3 -m venv /tmp/myjournal-a11y-venv
/tmp/myjournal-a11y-venv/bin/pip install playwright
curl --fail --location https://cdn.jsdelivr.net/npm/axe-core@4.10.3/axe.min.js -o /tmp/myjournal-axe.min.js
go build -o /tmp/myjournal-a11y-app ./cmd/app
/tmp/myjournal-a11y-venv/bin/python scripts/check-accessibility.py \
  --app-binary /tmp/myjournal-a11y-app \
  --axe-script /tmp/myjournal-axe.min.js \
  --report /tmp/myjournal-a11y-report.json
```

Axe is a QA-only tool; its [official documentation](https://github.com/dequelabs/axe-core) explains automated coverage and incomplete/manual review results. A passing scan is not full WCAG certification. Reports contain only synthetic fixture content, and should remain outside version control.
