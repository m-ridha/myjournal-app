# ADR 013: Autosave shares the conditional manual-save path

Date: 2026-10-09
Status: Accepted

## Decision

Use a five-second idle debounce after editor input, with no write for unchanged text. Manual Save, idle autosave, blur, and hidden/page-hide flushing all use the existing conditional PUT, snapshot acknowledgement, and account/date/tab-scoped recovery logic. There is at most one save in flight per editor. Text typed during a request stays dirty and backed up; after success, a new five-second debounce saves it against the returned version. Intentional empty edits are valid.

Restored drafts, migrated older drafts, and Keep both output require review: they do not automatically save on load, blur, or page-hide. Editing the reviewed text enables autosave; explicit Save remains available. Conflict controls must resolve a conflict before autosave can resume. Keep mine retains its explicit latest-version fetch and conditional save.

Any failed or ambiguous save pauses automatic writes for that editor, including blur/page-hide triggers. A successful manual Save or explicit successful conflict resolution resumes them; resolution output still waits for review. There is no automatic retry loop on network recovery or further typing; failed drafts and conflict actions remain available. Localized feedback explains automatic-saving pauses without replacing existing session/conflict/size/backup guidance.

Hidden/page-hide flushing uses fetch keepalive as a best effort. It cannot promise completion or acknowledgement during navigation, process termination, or browser keepalive quota exhaustion. Preserve the draft and beforeunload warning until an acknowledgement is processed; do not use sendBeacon or treat request dispatch as success. An in-flight save is never duplicated on page-hide. Sign-out notifications cancel queued autosaves and prevent late responses from clearing/recreating backups.

## Consequences

No new API, dependencies, offline queue, cross-tab synchronization, or server session storage is needed. A fresh tab and a recovered draft retain existing recovery semantics. A page restored from browser back/forward cache can process outstanding responses/timers normally; the same version and dirty guards apply.

The decision enables local autosave only while Google registration remains disabled. Before enabling autosave against Google storage, measure synthetic changed/unchanged saves and revision behavior as required by the Phase 1 S5 fallback. Drive revision history is not a recovery guarantee. A per-editor in-flight guard does not make Drive's checksum check/update atomic across clients.

## Manual retry after an uncertain save (2026-10-10)

An explicit Save after a non-conflict failed save first performs a fresh, no-store GET. If its valid entry text exactly matches the retry snapshot, use that observed text/version as acknowledgement without another PUT. Later typing stays dirty and backed up against the observed version. A valid absent entry can acknowledge an empty snapshot without creating a file. This verifies current content, not the historical outcome of the earlier request.

If saved text differs, submit the retry snapshot using its original base version; never silently rebase from the verification read. Normal conflict controls still apply. Failed, malformed or unauthorized verification makes no PUT, retains the draft/version and leaves autosave paused. Sign-out during verification prevents a later write. Only explicit retry performs this extra read; initial saves, ordinary autosave and explicit conflict-resolution fetches retain their existing flow. Reload already reconciles matching drafts through its initial GET. Acknowledged trash and successful explicit conflict resolution clear pending save-verification state.

This is best-effort reconciliation of an uncertain response, not atomic compare-and-swap or proof that another writer cannot change content immediately after a read. External Drive writers and multiple application instances remain subject to ADR 005's production gate. No retry loop or unconditional upload is introduced.
