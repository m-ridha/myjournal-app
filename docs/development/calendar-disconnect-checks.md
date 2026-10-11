# Calendar and disconnect checks

Use local development mode with synthetic entries. These checks do not validate Google revocation.

1. On Calendar, delay one month request, navigate to another month, and release the first response after the second succeeds. The newer label and grid must remain. Repeat with an older failed response and a delayed JSON body.
2. Fail the current month request. The grid must clear and show Calendar unavailable. Restore connectivity and select Today; the correct current month must return.
3. Save a synthetic entry, then make an unsaved change in an editor tab. Open Settings in another tab. Cancel Disconnect; the session and drafts must remain.
4. Hold the disconnect request. Both account buttons must be disabled. Return 503 or reject the request; the session and drafts must remain, and both buttons must become available again.
5. Allow disconnect to return 204. The app must return to the landing page, remove its session cookie, clear current-account backups, and stop writes in the other editor tab. Theme and unrelated-account backups must remain. Reconnect locally and verify the saved entry still exists.
6. Simulate storage cleanup failure after a successful disconnect. Retry cleanup; it must not issue another disconnect request. Unavailable storage before the request must preserve drafts and prevent disconnect.

Automated coverage: tests/browsing.test.js, tests/editor.test.js, and internal/driver/web/disconnect_test.go. The HTTP tests cover authenticated revocation, error mapping, cookie attributes, missing/expired sessions, and cross-origin rejection. Live Google revocation and reconnection remain separate acceptance work.
