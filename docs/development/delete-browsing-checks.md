# Entry deletion and browsing checks

Use synthetic past-date entries with the local development provider. Restart after rebuilding embedded assets. Existing journal content should remain untouched.

1. The red trash icon at the bottom right has a Move to trash tooltip/accessibility label, visible keyboard focus, and a 44px tap target. It floats over the bottom-right corner with a solid background on desktop and touch; there is no reserved bottom strip. Check both themes, long scrolled text, and resizing: right-side text padding and button inset keep text and the resize handle clear. On a missing date, Move to trash stays hidden. Save a synthetic entry; the action appears.
2. Type an unsaved change and click Move to trash. Cancel the confirmation. No DELETE or blur-triggered PUT should occur; the text and this tab's backup stay intact. Ordinary idle autosave remains active after cancellation.
3. Confirm Move to trash while a DELETE is delayed. Editor, Save, conflict actions, and Move to trash must be disabled/read-only. The backup remains until a 204 response. A failed/401/offline/404 response keeps text and backup, unlocks editing, and pauses automatic saves. Manual Save or another deletion attempt remains possible.
4. On successful trash, the editor becomes blank, this tab/date's backup clears, the status says Entry moved to trash, and the action hides. Other dates/accounts/tabs' backups remain. Verify the original saved file is in local trash; unsaved edits were explicitly discarded by confirmation. Typing again creates against an empty version. If browser backup removal fails, show the specific retained-backup warning.
5. While a PUT or conflict-resolution GET is pending, Move to trash must be disabled. Sign out in another tab during DELETE: late completion must not recreate drafts or permit more writes.
6. Seed more than 20 synthetic saved dates across years. Entries shows only the latest 20 rows, newest first, without Older/Newer controls. Its description mentions Calendar for older notes; Calendar remains accessible through the shared header. Successful loading leaves the status region empty; loading and failure feedback remain. Literal HTML in previews stays text. Test both English and Bahasa Indonesia.
7. Fail the Entries GET. Show an error and Retry; retry requests `/api/v1/entries?page=1&size=20` again. Old browser URLs such as `/entries?page=2` and arbitrary size parameters still request only the first twenty. The API's bounded pagination contract remains available separately.


Run `make test-js` for editor, timezone, and browsing regressions. Browser testing against local storage does not establish Google trash/folder behavior.
