# Editor recovery acceptance checks

Use the local provider and synthetic text. Stop your own spike first, then `make run` and sign in through the local development flow. The Google provider remains disabled. Automated checks (`make test-js` plus `go test ./...`) exercise mocked responses and rendering; these steps verify real browser storage, events, and network behavior separately.

1. Save a past-date entry with synthetic text A. Change it to B without saving, then reload and accept the browser's leave warning. B should be restored as dirty, even though A exists on the server. Save B and reload: B should load cleanly without a recovered draft.
2. Repeat with an intentionally empty or whitespace-only edit. Reload must retain it as an unsaved edit rather than falling back to nonempty server text.
3. Throttle requests in browser developer tools. Save B, then type C while the request is pending. Save should remain disabled during that request. When B's acknowledgement arrives, C must remain visible with neutral styling and `Unsaved changes`. A second save should send C against the new version.
4. Switch the browser network panel offline before saving a change. Save must become usable again after failure; the text and its backup should remain. Restore networking and retry. Do not share journal content or browser storage values in screenshots.
5. Use two tabs on the same saved date to trigger a stale-version conflict. The failed tab must keep its editor text and show an accurate conflict status. Reload must recover the stored draft and preserve its original base version rather than silently overwriting the server. Full conflict-resolution choices and same-date simultaneous-tab draft isolation remain separate acceptance work.
6. Clear only the app's session cookie while an editor tab has unsaved text, then save. The editor must preserve text and display the sign-in requirement. After reconnecting to the same local account, reopen the date and recover the draft.
7. Keep cookies enabled and simulate unavailable draft writes using the console steps below. Blocking all site data also blocks session cookies and cannot verify authenticated online saves. Saving should still work online. Failed-save status must not claim a local backup succeeded; it should advise keeping the page open or copying the text. Restore storage afterward.
8. With unsaved text, open `/settings` in another tab and click Sign out. Account-scoped drafts and legacy date-only drafts should be cleared, while theme preferences remain. Other open editors should stop recreating drafts after the sign-out storage event. If storage cannot be cleared, the settings page must explain that failure before completing sign-out.
9. An existing date-only backup from an earlier app version should offer `Review older draft`, without automatically assigning it to the current account. Only explicit confirmation restores it; migration should remove the legacy copy only after the scoped backup succeeds.
10. Delay or fail the initial GET. Save and typing should be disabled until an entry/version has loaded. The textarea is read-only from the initial HTML render; recovered drafts remain visible. Programmatic changes during loading must still be preserved. On failure, `Retry loading entry` should appear and recover safely.

Record browser/version and observed results without account identifiers, entry contents, or whole cookies. See PROGRESS.md and findings.md for reported browser checks. Actual phone testing was waived by the user in favor of device-toolbar checks. Debounced autosave, page-hide network saving, full conflict actions, and timezone correctness are outside this focused change.


## Storage-write failure (check 7)

Sign in normally, open a synthetic test entry, wait for loading to finish, then run in the console:

```js
window.mjOriginalSetItem = Storage.prototype.setItem;
Storage.prototype.setItem = function (key, value) {
  if (key.startsWith('mj-draft:')) throw new DOMException('Test storage failure', 'QuotaExceededError');
  return window.mjOriginalSetItem.call(this, key, value);
};
```

Type a change and Save online: it should succeed. Then type another change, switch network offline, and Save: the text must stay visible and the status must warn that backup is unavailable, without claiming the draft was kept locally. Restore networking and restore the method before retrying:

```js
Storage.prototype.setItem = window.mjOriginalSetItem;
delete window.mjOriginalSetItem;
```

Reloading also removes this temporary override. Do not disable all site storage for this check.

## Older draft recovery (check 9)

Use a disposable past entry, such as `/entries/2026-10-01`. Save its current text first so there is no unsaved scoped backup. In that tab's console, create a synthetic old-format backup:

```js
localStorage.setItem('mj-draft:2026-10-01', 'Synthetic older draft');
location.reload();
```

The server text should remain visible and `Review older draft` should appear. Click it and cancel: nothing should change. Click again and confirm: the synthetic text should appear as an unsaved restored draft. Verify the old key was removed:

```js
localStorage.getItem('mj-draft:2026-10-01') // null after successful migration
```

Reload: the synthetic text should now restore from the account-scoped backup. Save or clear the disposable entry afterward. This tests explicit migration of old backups whose account is unknown; it is not the normal draft recovery in checks 1–2.
