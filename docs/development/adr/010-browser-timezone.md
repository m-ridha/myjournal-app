# ADR 010: Browser time zone and fixed entry dates

Status: Accepted (2026-10-08)

The browser writes its IANA time zone to the `mj_tz` cookie. HTTP middleware resolves it once per request and passes the location through the domain context. Missing, invalid, oversized, and host-dependent `Local` values fall back to UTC. The app embeds Go's time-zone database.

Pages, calendar defaults/markers, and entry read/save/delete validation use the resolved location and injected clock. Future dates return the styled page 404 or API 400. This optional preference cookie contains no account or authentication data.

A synchronous browser bootstrap corrects a changed time-zone cookie before editor requests begin. A successful cookie update reloads the same date-dependent URL; blocked cookies do not create a reload loop. Default editor routes become `/entries/YYYY-MM-DD` before editing starts. Explicit selected dates, save targets, and draft keys never change when midnight passes. A page's displayed today remains its initial snapshot until navigation/reload; the server evaluates each subsequent request with its current clock.

Fixed-clock tests cover midnight, extreme UTC offsets, both New York DST transitions, future-operation storage isolation, and saves crossing midnight. Browser harness tests cover correction, canonical URLs, HTTPS cookie flags, and blocked-cookie fallback. Real Chrome/Safari/phone acceptance remains pending; configurable calendar week start is separate work.
