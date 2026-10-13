# ADR 022: Recover expired authentication without leaving the editor

- Status: Accepted
- Date: 2026-10-13

## Context

Protected JSON requests can return 401 when a cookie is missing, invalid or expired, Google access is unavailable, or a browser reconnects to a different account. Redirecting a fetch request does not navigate the browser page, and redirecting the editor can discard text when draft storage is unavailable. The home page may still render an authenticated editor when its cookie is valid but Google access has expired.

## Decision

A shared browser request helper shows a localized authentication-expired banner for protected entry, list, calendar and disconnect operations. The banner exposes a native link to `/sign-in` in a new tab, with an explicit new-tab label. It advises reconnecting to the same account and returning to retry. `/sign-in` always offers configured login providers, even with an existing cookie, and never clears the current session or drafts. It uses `Cache-Control: no-store` and `X-Robots-Tag: noindex`; it is omitted from the sitemap. Existing OAuth state, PKCE, nonce and cookie checks remain unchanged.

On editor expiry, cancel scheduled autosaves, pause automatic writes and preserve text and account/date-scoped backups. Manual save retry uses the existing fresh-read acknowledgement reconciliation and conditional version checks. Failed loads, saves, deletes and conflict reads retain their existing recovery semantics. Expiry never broadcasts successful logout or clears drafts; sign-out retains its separate acknowledgement and cleanup protocol. A successful explicit retry clears the banner. Stale calendar responses and successful requests predating the expiry do not dismiss or replace current feedback.

Editor and settings requests carry their rendered draft scope in `X-Myjournal-Account`. HTTP delivery compares it with the authenticated provider/account before invoking services. A mismatch returns the normal 401 envelope, preserving the old tab's text instead of writing or revoking under a newly selected account. The existing scope algorithm moves into authsession; credentials and account labels remain excluded. This optional header is an account-consistency guard, not authentication; ordinary API clients without it retain cookie authentication.

## Consequences

Recovery requires an explicit retry after reconnecting; no failed write is automatically replayed. The original editor stays open even if sessionStorage fails, and backup failures remain truthful. Closing the tab still expires its recovery storage under ADR 011. Reconnecting to a different account requires returning to the original account or explicitly preserving text before moving elsewhere. This change does not diagnose why a particular production request returned 401 or extend Google token/session lifetimes.
