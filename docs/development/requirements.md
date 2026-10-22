# Product requirements

myjournal is a quiet, online-only app for one plain-text entry per day. Production bridges to the user's Google Drive; local filesystem storage is a development exception. This guide owns product requirements; implementation choices are in [architecture](architecture.md).

## Entries and dates

- Store UTF-8 plain text with LF endings, without JSON wrappers, body metadata or rich-text rendering. Keep long text, emoji and line breaks readable.
- Past dates remain editable without a cutoff; future dates are rejected. Resolve URL-encoded IANA `mj_tz` once per request; missing, invalid, oversized or Local values fall back to UTC. Pages, calendar and future validation share the resolved location and injected clock, with embedded tzdata.
- Canonicalize default editor URLs to an explicit date. Correct stale timezone cookies before editing without reload loops when storage is blocked. Selected dates never shift at midnight; the loaded page's today marker is a snapshot until navigation/reload.
- Entries shows the latest 10 entries; Retry requests only API page 1 with the configured recent-entry limit. Calendar navigates older notes. Previews start at the first meaningful line after leading whitespace.
- Trash requires confirmation. A confirmed DELETE locks mutations until acknowledgement; only a 204 clears this tab/date backup. Ambiguous/failed trash retains text as unverified, and blur autosave must not preempt explicit actions.

## Editor and recovery

- The editor starts empty, with no sample text or placeholder, read-only and Save disabled until a valid GET. Failed loads offer Retry. Never clear text after a failed load/save, 401, conflict or ambiguous acknowledgement.
- Typing remains enabled during PUT. Only the acknowledged snapshot becomes clean; subsequent edits remain dirty and use the returned version. Save feedback says Saved, returning to neutral grey on editing.
- Autosave uses five-second idle debounce, suppresses unchanged content and permits one request in flight. Follow-up saves use acknowledged versions. Failures pause automatic writes until manual success or explicit conflict resolution. Blur/hidden/page-hide flushing is best effort and never clears unacknowledged backups.
- Account/date-scoped `mj-draft:v3` sessionStorage backups are independent per tab, survive reload/same-tab navigation and may expire when the tab closes. Scope excludes account labels and credentials. Empty/whitespace edits are drafts. Storage failures must not falsely claim backup success; advise keeping the page open or copying text.
- Legacy v2/date-only recovery requires explicit review, retains known base versions and removes the old copy only after successful backup migration. Recovered/migrated drafts and Keep both output wait for editing or Save before autosave.
- Manual retry after a failed save reads fresh current content first. An exact snapshot match can acknowledge without PUT; differing content retains the original conditional version. Failed verification keeps text and paused autosave. Later typing and sign-out guards remain protected.
- Conflict actions: Keep mine reads latest then conditionally saves; Load the other version confirms replacement and makes no PUT; Keep both appends saved text below local text for review. Further external changes can conflict again. Reconciliation establishes current content, not the historical outcome of an ambiguous write or atomic external-writer protection.

## Authentication and cleanup

Protected browser 401 responses show the shared sign-in-again banner and a new-tab `/sign-in` link. Preserve text/backups, pause autosave and require explicit retry after reconnecting to the same account. Expiry never means successful logout. Editor/settings requests send X-Myjournal-Account; HTTP rejects a scope mismatch before invoking services, preventing silent account switching.

Logout must succeed before deleting drafts or notifying other tabs. Failure preserves recovery and editor usability. Post-logout cleanup failure shows a truthful signed-out message and offers cleanup-only retry. Preserve other accounts and theme preferences. Suspended tabs and blocked browser storage limit cross-tab notification; do not promise immediate cleanup everywhere.

Settings Disconnect revokes the provider connection before clearing the session and reuses acknowledged sign-out cleanup. Failures retain session/drafts; cleanup-only retry never repeats revocation. Local disconnect preserves saved files. Sign out alone neither revokes Google access nor deletes saved entries.

## Interface and accessibility

- Header: brand, centered Write/Entries/Calendar, theme toggle, globe-and-EN/ID language picker, always-visible Settings gear and Help. Privacy is linked from Help. Settings/preferences are public; account data and sign-out/disconnect require a valid session.
- English is default; English/Bahasa Indonesia selection persists after sign-out in a browser preference cookie. Locale JSON uses flat dotted keys; templates use t and scripts data-messages. Dates, account labels and user text are runtime data.
- Theme follows device changes until the user explicitly chooses via the header. Respect saved choices. Do not restore Appearance controls in Settings.
- Footer independently centers Built for quiet writing.; wide screens place build version and fixed 2026 GitHub copyright at the right, narrow screens stack details on a centered second line. Local builds omit the version.
- Use visible keyboard focus, live-region feedback, disabled semantics, adequate touch targets, WCAG AA contrast, reduced-motion support and 200% zoom. Color alone cannot convey status/calendar state. Preserve narrow-screen wrapping.
- Calendar month-heading controls open localized 12-month/12-year grids without entry indicators or data requests while browsing. Disable future choices/next-month at request-resolved today, preserve year-one bounds, keyboard/focus dismissal, and latest-request-only rendering including failures.
- No third-party page requests, remote fonts, trackers or CDN assets. Journal text is never trusted as HTML.

## Quality targets

Target at least 85% coverage for each maintained production function; generated bodies and test support are excluded. Package-level coverage does not establish this per-function goal. Performance targets at p75 are today’s load under 1.5 seconds, manual save under 1 second and cached calendar navigation under 500 ms. These remain goals, not passed acceptance; see [status](status.md#outstanding-work).

## Scope and accepted constraints

No offline synchronization, extra storage providers, attachments, Markdown, client-side encryption, search, tags, sharing, reminders, AI summaries, native apps or journal-content analytics. Redis remains deferred. Monday is the fixed calendar week start; configurability is shelved. Closed-tab draft expiry and device-toolbar coverage instead of physical-phone testing are accepted for this stage.

Google Console publication is In Production (public) with verified branding. Preserve one production replica and avoid simultaneous external editing; process/deployment overlap, non-atomic writes, token expiry and latency/revision limitations are detailed in [status](status.md).
