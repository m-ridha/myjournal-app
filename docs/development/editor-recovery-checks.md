# Editor recovery acceptance checks

Use the local provider and synthetic text. Stop your own spike first, then `make run` and sign in through the local development flow. The Google provider remains disabled. Automated checks (`make test-js` plus `go test ./...`) exercise mocked responses and rendering; these steps verify real browser storage, events, and network behavior separately.

1. Save a past-date entry with synthetic text A. Change it to B without saving, then reload and accept the browser's leave warning. B should be restored as dirty, even though A exists on the server. Save B and reload: B should load cleanly without a recovered draft.
2. Repeat with an intentionally empty or whitespace-only edit. Reload must retain it as an unsaved edit rather than falling back to nonempty server text.
3. Throttle requests in browser developer tools. Save B, then type C while the request is pending. Save should remain disabled during that request. When B's acknowledgement arrives, C must remain visible with neutral styling and `Unsaved changes`. A second save should send C against the new version.
4. Switch the browser network panel offline before saving a change. Save must become usable again after failure; the text and its backup should remain. Restore networking and retry. Do not share journal content or browser storage values in screenshots.
5. Use two tabs on the same saved date to trigger a stale-version conflict. The failed tab must keep its editor text and show an accurate conflict status. Reload must recover the stored draft and preserve its original base version rather than silently overwriting the server. The conflict actions are now available: Keep mine, Load the other version (confirm/cancel), and Keep both (review, then Save). Retest them using the steps below.
6. Clear only the app's session cookie while an editor tab has unsaved text, then save. The editor must preserve text and display the sign-in requirement. After reconnecting to the same local account, return to the original tab or reopen the date in that same tab and recover the draft. A fresh tab does not share its backup.
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


## Conflict actions and independent tab backups

New backups use tab-local sessionStorage. Reload and same-tab navigation retain them. Closing the tab can discard them, as agreed with the user; old local backups remain available through Review older draft.

1. Open a saved synthetic past entry in tabs A and B. Change each to distinct text without saving. Reload each: it must recover its own text. Saving A must not clear B's backup.
2. Save A, then Save B to get a conflict. Reload B: it should retain B's text and offer the conflict actions again.
3. Choose Keep mine in B: it should save B's text against a freshly fetched version. Reload A: A must load B's saved text if A has no unsaved backup.
4. Create another conflict. Choose Load the other version and cancel: your text stays. Confirm it: saved text replaces your draft, with no PUT request.
5. Create another conflict. Choose Keep both: your text comes first, followed by a blank line and the latest saved text. It stays Unsaved changes until you review and Save.
6. Throttle conflict-resolution requests or go offline: failed resolution must retain your text and allow retry. Changes by another writer between the latest GET and the PUT must still produce a conflict. Typing while the Keep mine PUT is pending must stay unsaved afterward.
7. Sign out from Settings in one tab: other open editors must stop saving/recreating their backups. Verify this account's tab backups are cleared; theme preference remains.
8. Use the new Settings header link on desktop and device-toolbar widths. Appearance is removed from Settings; the shared header theme button still works.

Record observations here or in findings.md. The user confirmed all eight checks above passed on 2026-10-09; results are recorded in findings.md. Browser/version was not specified in that report.


## Failed logout recovery (PR #3 follow-up)

1. Leave synthetic unsaved text in an editor tab. Open Settings in another tab.
2. Set Settings' network offline, then click Sign out. It should show Unable to sign out and allow retry. The original editor must remain editable, keep its leave warning and backup, and recover its text on reload. No final sign-out notification should be written for this failed attempt.
3. Restore networking. The editor should still be able to Save if the failed request never reached the server. Retry Sign out: only confirmed success should clear backups and stop editing in other tabs.
4. If testing HTTP 503 with request interception, repeat the backup-retention check with that response. A transport failure can occur after the server has cleared the cookie; in that case text still recovers, but saving can require reconnecting.

Automated regressions cover pending logout, HTTP 503, network rejection, editor recovery/save after failure, successful retry, and a storage failure after successful logout. The latter must clearly say signed out and offer Retry draft cleanup without repeating the logout request. The user confirmed this failed-logout recovery checklist works on 2026-10-09; browser/version and any optional request-interception scenarios were not specified.


## Oversized save recovery (PR #5 follow-up)

Use a disposable past entry. Copy any existing unsaved text somewhere safe first. After the entry finishes loading, run this in the browser console to insert synthetic oversized text:

```js
const oversizedEditor = document.querySelector('[data-entry-text]');
oversizedEditor.value = 'x'.repeat(1024 * 1024);
oversizedEditor.dispatchEvent(new Event('input', { bubbles: true }));
```

1. Click Save. The PUT must return 413. The status should identify the 1 MiB request limit and advise copying the full text somewhere safe before shortening it. The text stays visible/editable and the page must not show Saved.
2. If browser backup is available, the status also says Draft kept locally. Reload: the synthetic draft should restore. If quota/storage access prevents backup, the status must instead advise keeping the page open or copying the text; reload recovery is not promised.
3. After copying anything you need to preserve, replace the synthetic text with a short test sentence and Save. It should succeed and clear the saved draft.

The cap applies to the entire serialized JSON request, so exactly 1 MiB of ASCII text exceeds it once the request fields are added. Automated regressions cover the 413 response, newer edits during the pending request, recovery/retry, and unavailable backup. This browser follow-up is not yet human-reported.


## CSP and security-header browser check

1. Restart the app. In Chrome and Safari, open the landing page and sign in locally. Network response headers should include Content-Security-Policy and Cache-Control: no-store for pages/API responses. The console should show no app-caused CSP violations.
2. Toggle light/dark, reload, and confirm the selected theme and gear icons still render. The external theme-init.js should load before CSS. If browser storage is blocked, the app must still load; stored theme recovery is not promised.
3. Open a disposable past entry, type synthetic text, Save, then verify list preview and calendar navigation. Existing draft/conflict behavior must remain intact.
4. Open Settings and sign out. Confirm the normal sign-out/draft cleanup behavior. Check a styled missing page and signed-out API rejection for security headers as well.

Automated browser verification on 2026-10-09 passed the covered flows in Chrome 155.0.8059.40 (headless) and native Safari 26.4, including deliberate inline-script blocking, conflict/Keep mine, draft recovery, and logout cleanup. Safari full startup-console history and Safari blocked-storage behavior remain unverified. See [browser QA evidence](csp-browser-qa.md) for scope, expected negative probes, and limitations.

## Uncertain-save manual retry

Automated interaction regressions pass. Agent-driven Chrome/local-provider validation passed on 2026-10-10 with scoped synthetic fetch failures/delays; see the evidence below. Use a disposable synthetic past entry, with other tabs idle. Never use personal text or the retained January 3 fixture.

1. Simulate one lost acknowledgement without preventing the real save: in DevTools on the synthetic editor, replace `fetch` temporarily with the following wrapper, then type distinct synthetic text and click Save. It restores fetch as soon as the next entry PUT responds. The real PUT still runs; the editor receives a deliberate rejection. If you abandon the test before a PUT, run `window.fetch = originalFetch` to restore it.

   ```js
   var originalFetch = window.fetch;
   window.fetch = async function (input, init) {
     const response = await originalFetch.call(window, input, init);
     if (typeof input === 'string' && input.startsWith('/api/v1/entries/') && init?.method === 'PUT') {
       window.fetch = originalFetch;
       throw new Error('Synthetic lost acknowledgement');
     }
     return response;
   };
   ```

2. Confirm the page retains its text and draft with failed-save feedback. Wait more than five seconds: no automatic retry should occur. In Network, click Save again: expect one GET, no additional PUT, and Saved if content already matches.
3. Repeat, but type newer text before retrying. The verification GET must not silently adopt a newer server version for different text; a conditional PUT using the original version should conflict if the earlier upload committed. Use the existing conflict controls to choose the desired result.
4. Throttle the verification GET, click Save, and type while it is pending. A matching snapshot acknowledgement must preserve later typing as Unsaved changes, with its backup and the verified version. Save remains disabled during verification; typing remains enabled.
5. After another simulated failure, switch offline before retrying. Verification failure must make no PUT, retain text/backup/version and pause automatic writes. Reconnect and manually retry. Sign out in a separate tab while verification is pending to confirm no late write or draft recreation.

The wrapper is a synthetic response-loss test, not evidence of a Google timeout cause. Trash the disposable entry only after successful reconciliation and review.


### Observed local browser evidence — 2026-10-10

Executed with the Web App QA & Release Copilot skill using Chrome on macOS, localhost:3030 and the local development account. Exact Chrome version was not captured. Used an initially blank February 20, 2020 entry. A DevTools fetch wrapper affected only that exact entry API path, recorded method names only, and injected lost acknowledgements, a delayed GET response and a rejected verification GET. The offline case simulates a network rejection; browser-wide Offline mode was not toggled. No Google access or external-writer atomicity is established by this run.

| Check | Observed evidence | Result |
| --- | --- | --- |
| Lost save acknowledgement | Real PUT completed; injected rejection displayed Save failed. Draft kept locally. Text and tab backup remained. | Pass |
| Paused automatic retry | More than five seconds elapsed during inspection; method trace remained one PUT. | Pass |
| Matching manual retry | Method trace PUT, GET; no duplicate PUT. UI changed to Saved. | Pass |
| Newer text before retry | Trace PUT, GET, PUT; conditional retry produced conflict controls with newer draft intact. Keep mine subsequently saved it. | Pass |
| Typing during held verification | Saving and disabled Save while GET held; textarea remained editable. Releasing matching snapshot showed Unsaved changes with later text intact; subsequent autosave reached Saved. | Pass |
| Verification network failure | Trace PUT, GET with no further PUT; draft backup present and Save available. Returning probe to normal and explicitly retrying reached Saved. | Pass |
| Sign-out during verification | Separate Settings tab signed out while GET was held. Releasing it left trace PUT, GET, disabled signed-out editor and no fixture draft. No late PUT or backup recreation. | Pass |
| Cleanup | Restored native fetch, signed in locally, trashed only the disposable entry via confirmation. Reload showed Blank page, empty editor and no trash control. | Pass |

The initially attempted paste did not install the probe; its ordinary successful save is excluded from failure evidence. Console input was then verified before running the cases above. Extension-origin favicon CSP messages were visible and are not attributed to app behavior. Agent-created QA tabs are temporary; local sign-in was restored after the sign-out test. No product code changed during QA. Safari and Google-backed repetition remain separate coverage.
