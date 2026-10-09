# Safe autosave checks

Use the local development provider and synthetic past-date entries. Keep existing journal text untouched. Run `make run`, sign in with local development, and open a test date. Server changes/assets are embedded: restart after rebuilding changes.

1. Type and keep focus in the editor. Before five seconds there should be no PUT; after five seconds idle, one PUT and Saved. More typing resets the interval. Leaving the text unchanged produces no periodic writes. Reverting to the acknowledged text produces no write.
2. Delay a PUT in browser network tools. Type more while Saving. No second PUT starts concurrently. When the first response succeeds, newer text remains Unsaved changes and backed up. After another five seconds it saves using the returned version.
3. Go offline, edit, and wait. Save failed, Draft kept locally, and Automatic saving is paused should appear. Text remains editable and Save becomes available. More typing, blur, and waiting must not retry automatically. Return online and use Save; after success, later edits autosave again.
4. Repeat with a mocked 401, 413, and 503 or failed response. Keep session/size guidance and the draft. When browser storage is unavailable, show the truthful backup-unavailable warning rather than Draft kept locally.
5. Reload an unsaved backup. Draft restored should remain until review; waiting, blur, and navigation must not automatically save it. Edit to enable autosave or explicitly Save. Repeat with older-draft migration.
6. Open the same date in two tabs. Save a change in A, then edit B and wait for its autosave. B must show conflict controls and preserve its own backup. Typing/waiting must not bypass the conflict. Check Keep mine, confirmed Load the other version, and Keep both. Keep both must stay unsaved for review until editing or Save.
7. Type and click outside the editor before the idle timeout. Blur should flush once. Switch tabs or navigate away: flushing is best effort, with recovery retained until acknowledgement. If a PUT is pending, navigation must not cause another concurrent PUT; the newest local text remains backed up.
8. Save intentionally empty text. Verify an empty entry is acknowledged and the backup clears. Sign out from Settings in another tab while an idle timer or PUT is pending: no subsequent automatic writes or recreated backups.

Automated timer and state regressions are in `tests/editor.test.js`; run `make test-js`. Those tests do not prove browser navigation delivery or Google behavior. Keep `editor-recovery-checks.md` as the adjacent regression checklist; do not assume historical manual results cover this new behavior.
