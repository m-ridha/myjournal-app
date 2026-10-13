# Focused UI acceptance checks

Covered on 2026-10-09 in Chrome 155.0.8059.40 with isolated local-provider data:

- With no mj-theme preference, initial device dark/light is applied before CSS and device changes update the page. Space on the theme toggle chooses explicitly; later device changes cannot override it and reload preserves it.
- Keyboard Save accepts long plain text, emoji, accented characters, many lines, and a long unbroken sequence; wrapping does not lose text.
- Editor, Entries, Calendar, Settings, Help, Privacy, and styled 404 render in both themes at 390px and at 200% CSS zoom without horizontal page overflow. Visible empty-text buttons have accessible labels.
- Reduced-motion media preference removes calendar transitions. Editor status retains its live region; the future-day control is disabled.
- No browser page exceptions during those flows.

Reproduce using synthetic text and a disposable local data directory. Automated theme state regressions are in tests/theme.test.js and run via make test-js. Architecture checks run separately via make test-architecture; its temporary probes must not overlap another build/lint process.

Current focused UI acceptance status (2026-10-11): the requested Chrome native-zoom, keyboard/focus, sampled non-text contrast, basic human-reported screen-reader announcement and external-icon provenance checks have evidence below. The final continuation covers the previously outstanding local error/calendar/confirmation review. No runtime defect was found. This closes this focused checklist; it is not full WCAG certification or a declaration that every Phase 2/3 requirement is complete.

Actual Google folder links, revocation and authentication remain provider acceptance, not local UI results. Native screen-reader versions and the origin of the additional “Network data available” announcement are not established.

The physical-phone waiver remains in effect; narrow Chrome viewport coverage is not physical-device evidence. Native Safari is not included in this fresh focused run.

Resolved architecture decision: the user explicitly retained JSON API and vanilla JavaScript and excluded HTMX (ADR 014). The HTMX assessment remains historical research, not an open acceptance gate.

## Automated accessibility follow-up (2026-10-10)

Chrome 155.0.8059.40 and axe-core 4.10.3 passed 18 WCAG A/AA-tagged scans: seven routes plus failed-save and conflict states in both themes. The audit initially reproduced a dark Settings Disconnect text contrast failure (2.55:1) and a calendar naming issue. Using the existing danger theme variable and adding role=group to the named calendar container resolved them. All visible enabled links/buttons/editable textareas were reached by Tab and had a computed keyboard focus indicator on the seven routes and conflict editor. This checks reachability/indicators, not the subjective quality of every focus order or state.

The scanner leaves arrow-glyph contrast as incomplete. Supplemental computed foreground/background measurements for enabled previous/next controls passed 4.5:1, with observed ratios 13.98–17.40:1. These measurements do not certify all non-text control boundaries or focus-indicator contrast. Incomplete results remain in the report instead of being discarded. Native screen-reader and native zoom checks remain open.

The optional scripts/python/check-accessibility.py tool uses an owned ephemeral-port server, temporary synthetic local data, and a random session key. It shuts down the server and removes data. It is separate from ordinary CI and adds no runtime dependency or page request to a CDN. Supply a locally downloaded axe script; do not run this against real journal data.

Example setup/run (requires Python, installed Chrome, and network access for initial QA tools):

```sh
python3 -m venv /tmp/myjournal-a11y-venv
/tmp/myjournal-a11y-venv/bin/pip install playwright
curl --fail --location https://cdn.jsdelivr.net/npm/axe-core@4.10.3/axe.min.js -o /tmp/myjournal-axe.min.js
go build -o /tmp/myjournal-a11y-app ./cmd/app
/tmp/myjournal-a11y-venv/bin/python scripts/python/check-accessibility.py \
  --app-binary /tmp/myjournal-a11y-app \
  --axe-script /tmp/myjournal-axe.min.js \
  --report /tmp/myjournal-a11y-report.json
```

Axe is a QA-only tool; its [official documentation](https://github.com/dequelabs/axe-core) explains automated coverage and incomplete/manual review results. A passing scan is not full WCAG certification. Reports contain only synthetic fixture content, and should remain outside version control.

## Native browser and keyboard follow-up (2026-10-11)

Observed in Chrome using a separate local-provider server on localhost:3032 and disposable synthetic files. No real journal or Google account data was used. No runtime code changed in this follow-up.

- Chrome's native toolbar reported 200% zoom. Read-only page measurements confirmed viewport width 756, devicePixelRatio 4 and CSS zoom 1 (baseline width 1512/devicePixelRatio 2). This replaces the outstanding native-zoom probe, rather than inferring browser zoom from CSS.
- Signed-out Home, Privacy, Help, Terms and styled 404, and signed-in Editor, Entries, Calendar, Settings, Privacy, Help and Terms had document width equal to viewport width in both themes. Editor/conflict screenshots were inspected at this zoom; this does not certify every page pixel or every viewport.
- Tab traversed enabled controls on Entries (including Older pagination), Calendar (including Today/month controls and enabled dates), Settings, Privacy, Help and Terms in both themes. All observed controls entered the viewport and retained the browser focus outline. Editor traversal in dark mode reached date controls, Save, textarea and trash in order; textarea uses its wrapper focus treatment and trash uses its own outline. Conflict buttons were reached in order in dark mode; light Load the other version retained its native outline. This is observed reachability and partial visual evidence, not an exhaustive focus-obscuration certification.
- External modification of a synthetic file produced the conflict message while preserving the local draft. Keep both retained both texts for review and subsequent Save acknowledged them; a second conflict resolved through Keep mine. Both-theme conflict layouts had no horizontal overflow. Load the other version was reached but its replacement confirmation was not exercised in this follow-up.
- Temporarily replacing the disposable entries directory with a file induced a real local I/O save failure. The editor retained the exact synthetic text and displayed `Save failed. Draft kept locally.` in its polite status region. Restoring the fixture and manually retrying returned `Saved`. No page console warnings/errors were captured during the sampled flow. Accessible-tree/status inspection is not evidence of spoken announcements.
- Supplemental WCAG luminance calculations from inspected theme colors: control border #767676 against white/#f5f5f5 is 4.54/4.17:1; dark #8a8a8a against #121212/#1e1e1e is 5.43/4.83:1. Trash glyph/border #b3261e on #ededed is 5.58:1 and #ff9b94 on #262626 is 7.47:1. Editor focus/text #1a1a1a on #f5f5f5 and #ebebeb on #1e1e1e is 15.96/13.98:1. These sampled custom indicators exceed 3:1; native browser focus rendering and all calendar states are not comprehensively certified by these calculations.

Full UI acceptance remains open: native screen-reader Save/failure/conflict announcements need human observation; exhaustive error-state/focus and calendar non-text review remain; external trash-icon provenance is unresolved. A human screen-reader check was requested. No new axe scan, Safari/physical-device result or Google-provider acceptance is claimed here.

## Human screen-reader and icon follow-up (2026-10-11)

The user reported that Save announces `Saved`, failed save announces the failure, and conflict announces the conflict and exposes Keep mine / Load the other version / Keep both. They also reported `Network data available` during a failed-save attempt. Preserve both observations: the latter wording is not the application's save-failure string, and its origin/order was not independently established. Browser, screen-reader product/version and exact failure setup were not supplied; these are human-reported results, not agent-observed speech or a complete screen-reader matrix. The requested basic announcement check has human confirmation; broader state/focus/calendar acceptance remains open.

The user supplied the trash icon source, SVG Repo #335255. Its source page lists CC0; ADR 004 records the source, license and failed direct-download comparison. The missing source/license gate is resolved on that evidence, without replacing the icon.

## Final focused UI continuation (2026-10-11)

Preserved the user's screen-reader test tab and draft by using a separate tab. Reused only the disposable localhost:3032 provider/data. Restored every temporarily changed fixture directory, viewport override and native browser zoom afterward.

- Induced real editor-load and page-2 list failures by temporarily replacing the synthetic entries directory with a file. Both themes exposed Retry, preserved the disabled Save/read-only initial-load boundary, and allowed keyboard access with no observed header obstruction. The read-only textarea remained focusable with its wrapper focus treatment. The list retained its Newer link and same-page Retry.
- Induced a real calendar failure by temporarily replacing the synthetic 2026 directory with a file. Both themes displayed Calendar unavailable; Today and month controls retained keyboard focus indicators without obstruction. Restoring the directory and activating Today recovered the current month and saved-entry marker. Earlier calendar probes while the root fixture was changed continued to render cached data and are not counted as failure evidence.
- Inspected calendar saved-entry dot and double/inset today outline in both themes. Saved-entry dot contrast is 15.96:1 light / 13.98:1 dark; today outline contrast is 4.17:1 light / 4.83:1 dark against the day surface. Shape and accessible labels distinguish today/has-entry; future/outside-month cells are disabled and named. Low-contrast inactive cells are not enabled controls. Ordinary day boundaries are decorative: high-contrast date labels identify the enabled controls. Native browser outlines remain in place on focused dates.
- Load the other version opened the explicit native replacement confirmation. Cancel retained the synthetic local draft and returned focus to its action. Keyboard activation followed by acceptance loaded the exact external synthetic version. Trash opened its explicit confirmation; Escape retained the entry and returned focus to trash, and acceptance produced Entry moved to trash with an empty editor. Only the purpose-created 2019-02-01 fixture was trashed; the user's screen-reader fixture was preserved.
- Repeated keyboard traversal and no-horizontal-overflow checks on Editor, page-2 Entries, Calendar, Settings, Privacy, Help, Terms and styled 404 in both themes at 390×844, then native browser 200% (756 CSS px, devicePixelRatio 4, CSS zoom 1). Focused controls remained at least partially visible outside the sticky header; no inaccessible control or focus trap was observed. The editor's tall textarea can extend beyond the viewport, while its focus treatment remains visible. Native zoom restored to 100% and viewport override reset.
- Fresh make test-js passed all five suites (editor, timezone, browsing, theme, benchmark). git diff --check passed. No runtime source edits, new axe run, deployment, commit or push. Historical automated accessibility evidence and human screen-reader observations remain explicitly separate from this run.

Result: the focused local Chrome UI acceptance checklist is complete with the recorded human announcement evidence and device-testing waiver. Full cross-browser/assistive-technology certification and full Phase 2/3 completion are not asserted.

## Expanded automated QA matrix (2026-10-11)

scripts/python/check-accessibility.py now includes signed-in /terms and a separate signed-out browser context for Home, Help, Privacy, Terms and styled 404 in both themes. Public-page checks assert expected status, no login redirect and no signed-in Settings link; Home must expose Local Dev login. The defined matrix is 30 axe scans: eight signed-in routes plus failed-save/conflict states per theme (20), plus five signed-out routes per theme (10). Both route matrices run keyboard checks; conflict retains its existing keyboard check. This is planned executable coverage, not a claim that 30 fresh scans passed. Script syntax was checked; the expanded Playwright/axe browser run was not executed in this task. Earlier 18-scan results remain historical evidence.

## Expanded automated matrix execution (2026-10-11)

Built the current app with go build -o /tmp/myjournal-a11y-app ./cmd/app and ran the expanded script using the existing local axe file and Python Playwright environment:

```sh
/tmp/myjournal-a11y-venv/bin/python scripts/python/check-accessibility.py \
  --app-binary /tmp/myjournal-a11y-app \
  --axe-script /tmp/myjournal-axe.min.js \
  --report /tmp/myjournal-a11y-expanded-20261011.json
```

Fresh Chrome 155.0.8059.40 / axe-core 4.10.3 result: **30 scans, zero violations, 28 route/state keyboard checks, zero page errors**. Signed-in Terms and all ten signed-out theme/route combinations passed, including the public status/no-redirect/no-Settings and homepage login assertions. Failed-save states were scanned; keyboard checks cover conflict and normal route matrices, not failed-save states in this script.

Axe retained color-contrast incomplete results for non-text previous/next arrow glyphs. Inspected the incomplete nodes: they concern date/month arrows, not new Terms or signed-out page failures. The separate computed enabled-arrow measurements passed at 13.98–17.40:1. Keep incomplete findings in the JSON report rather than treating them as automated axe passes. Native speech, physical-device and full WCAG certification are outside this automated run.

The script used its owned ephemeral-port server, random session key and temporary synthetic data and exited successfully with server/data cleanup. The JSON report remains outside version control. The earlier “not executed” note describes preparation before this successful run; it is superseded by this execution evidence.

### Public-header accessibility regression (2026-10-12)

scripts/python/check-accessibility.py now expects the Settings gear in both session states, adds signed-out Settings to both themes, checks that account actions are absent there, and scans the open header language menu. Fresh local execution passed 34 axe scans with zero violations, 30 keyboard-focus checks and zero page errors. Report: /private/tmp/myjournal-header-a11y-20261012.json. Axe retains incomplete aria-controls checks and sampled contrast checks; direct DOM assertions confirmed each controlled ID is unique and references a listbox, open menus render below their trigger, and computed header-language-button contrast exceeds 4.5:1 in both themes. The existing arrow contrast assertions also passed. These checks do not establish native screen-reader announcements or full WCAG conformance. Historical “no signed-in Settings link” coverage above is superseded by the user-approved always-visible public Settings navigation.

## Authentication-expiry follow-up (2026-10-13)

The optional QA script now covers `/sign-in` in both session states, expired Entries/Calendar/editor requests, and reconnecting through a new tab without losing the original editor. Current results and limits are recorded once in [authentication-expiry evidence](findings.md#authentication-expiry-recovery-2026-10-13). Native screen-reader announcements and Google reconnect still require separate human/live evidence; no prior acceptance gate is reopened here.
