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
- A measured WCAG AA contrast audit of text, controls, states, and focus in both themes; the current checks are not a full accessibility certification.
- Verified theme source/version/license provenance. ADR 004 records the intended Monochrome port but does not establish which assets were copied.

- Actual Google folder links, revocation, and authentication remain provider acceptance, not local UI results.

The physical-phone waiver remains in effect; narrow Chrome viewport coverage is not physical-device evidence. Native Safari is not included in this fresh focused run.

Resolved architecture decision: the user explicitly retained JSON API and vanilla JavaScript and excluded HTMX (ADR 014). The HTMX assessment remains historical research, not an open acceptance gate.
