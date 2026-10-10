# Google app integration checks

This tests the actual app, not the accepted standalone spikes. Use synthetic text and record results without tokens, cookies, authorization codes, account claims, or journal content. Automated checks do not constitute live acceptance. Production remains disabled; localhost test mode explicitly permits exercising the unfinished acceptance gates, including the external-writer race.

## Configuration and start

The user confirmed on 2026-10-10 that the Google Drive API, test user, and Web application client redirect are configured. The authorized redirect must be exactly:

`http://localhost:3030/api/v1/auth/google/callback`

Keep existing spike callback URIs if still using them. Keep only `openid`, `email`, `profile`, and `https://www.googleapis.com/auth/drive.file`. Do not broaden scopes to see manually created folders. This integration creates a separately app-tagged `myjournal` tree, not an S02 spike tree.

In your private `.env`, retain the existing valid SESSION_KEY and OAuth client values, and set:

```dotenv
APP_BASE_URL=http://localhost:3030
PORT=3030
PROVIDERS=google
GOOGLE_TEST_MODE=true
COOKIE_SECURE=false
```

Then `make run`. Stop an existing app/spike yourself before reusing port 3030. Do not rotate SESSION_KEY between restart checks. HTTP localhost uses insecure cookies for Safari compatibility; production requires HTTPS/Secure cookies and is not permitted by this test switch. Do not publish the Console app or alter billing for these tests.

To switch back: stop the process, set `PROVIDERS=local` and `GOOGLE_TEST_MODE=false`, then restart. Google and local connections are separate. Do not copy session cookies between them.

## Integrated acceptance

1. Open `/api/v1/auth/google/login?force_consent=1`, approve the four permissions, and confirm return to an editable dated editor. Check `mj_session` is HttpOnly, SameSite=Lax, Path=/, around 30 days, and below 3800 bytes including attributes. Pending `mj_oauth` should be removed. Do not share its value.
2. Write synthetic multiline text with Unicode and CR/LF input on a chosen past date. Verify Save, then reload. In Drive, inspect/download the exact UTF-8/LF plain text at `myjournal/entries/<year>/YYYY-MM-DD.txt`, with no wrapper or metadata in its body. The app discovers only files accessible under drive.file.
3. Check Entries previews, Calendar markers, past-date edits, and Settings's folder link after the first save. A fresh account's read/list/calendar/settings operations must not create folders.
4. Stop and restart with the same key. Reload the entry: only the refresh token is in the cookie, so a successful post-restart Drive read exercises fresh access-token acquisition. Repeated reads should work; cache request counts are covered by automated tests rather than exposing tokens in the UI.
5. With the existing Google session intact, open `/api/v1/auth/google/login` without forced consent. Confirm login and a subsequent restart/read succeed if Google omits a refresh token. A login without a refresh token and without a same-account existing session is rejected; retry with `?force_consent=1`.
6. Cancel consent, or decline the Drive grant when Google offers granular permissions. Confirm login fails and an existing session/draft remains intact. To test a different Google account, sign out explicitly first: same-provider reconnect rejects an unexpected account switch to avoid attaching drafts to another account.
7. Rename the app's root/entries/year folders; move the year folder elsewhere. Reload/read/save and confirm tagged discovery still uses the existing folders. Trash an empty test year/entries folder and verify the next write recreates missing paths; reads must not recreate them. Keep any duplicate/data-bearing folders: selection is deterministic, not a merge or deletion policy. Use synthetic data only.
8. Open the same date in two tabs. Save A, then save stale B; verify conflict and retained text. Exercise Keep mine, Load the other version (cancel and confirm), and Keep both. Repeat with a saved file edited or trashed in Drive. Creating/deleting elsewhere and another writer during resolution must preserve text and may cause another conflict.
9. Trash through the app with confirmation. Verify the file is in Drive Trash, then verify a subsequent new entry for that date has a different file ID; older trash remains preserved.
10. Introduce unsaved synthetic text, revoke the app's third-party connection in your Google account, restart the app to discard cached access tokens, then Save. Expect 401/reconnect behavior and retained draft. Open `/api/v1/auth/google/login?force_consent=1` in another tab, reconnect using the same account, and retry Save in the original tab. Keep the original editor open throughout. Revocation may not invalidate cached access tokens immediately; restarting makes the refresh check deterministic.
11. Test Settings Disconnect. Google revocation must finish before sign-out/draft cleanup; failures retain the session/draft. A subsequent sign-in may require consent. A plain Sign out does not revoke Google or trash saved files.
12. Test offline/failed requests and reload recovery using the existing recovery/autosave checklist. Examine logs: method, query-free path, status, duration, request ID only; no callback query, credentials, cookie contents, or journal text.

## Revision probe before accepting Google autosave

Do this only on a spare past date with no existing entry, from Settings (so an editor does not autosave concurrently). The probe refuses to overwrite an existing entry. It makes 50 changed saves, followed by 50 unchanged saves, using synthetic content. Keep Drive's file Manage versions panel available and record its displayed revision count before/after each stage, file size, date/browser, and whether Drive coalesces or purges revisions. Do not infer retention guarantees from one run. Do not run two probes concurrently.

Paste into DevTools Console on your localhost app:

```js
window.mjRevisionProbe = async (date, stage) => {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) throw new Error('Use YYYY-MM-DD');
  const settingsResponse = await fetch('/api/v1/settings');
  if (!settingsResponse.ok) throw new Error('Sign in first');
  const settings = await settingsResponse.json();
  if (settings.provider !== 'Google') throw new Error('Google session required');
  const endpoint = '/api/v1/entries/' + date;
  const response = await fetch(endpoint);
  if (!response.ok) throw new Error('Initial read failed: ' + response.status);
  let entry = await response.json();
  const marker = 'myjournal synthetic revision probe';
  if (stage === 'changed' && entry.exists) throw new Error('Choose an empty date');
  if (stage === 'unchanged' && (!entry.exists || entry.text !== marker + '\n50'))
    throw new Error('Complete the changed stage on this date first');
  if (!['changed', 'unchanged'].includes(stage)) throw new Error('Unknown stage');
  const start = performance.now();
  let versionChanges = 0;
  for (let i = 1; i <= 50; i++) {
    const text = stage === 'changed' ? marker + '\n' + i : entry.text;
    const saved = await fetch(endpoint, {method: 'PUT',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({text, baseVersion: entry.version || ''})});
    if (!saved.ok) throw new Error('Stopped at save ' + i + ': HTTP ' + saved.status);
    const next = await saved.json();
    if (!next.exists || next.text !== text || !next.version) throw new Error('Invalid acknowledgement');
    if (next.version !== entry.version) versionChanges++;
    entry = next;
    await new Promise(resolve => setTimeout(resolve, 250));
  }
  return {stage, saves: 50, versionChanges,
    elapsedMs: Math.round(performance.now() - start)};
};
```

Run `await mjRevisionProbe('YOUR-UNUSED-PAST-DATE', 'changed')`, inspect Drive versions, then run the same date with `'unchanged'`. Expect 50 changed acknowledgements and 0 checksum changes in the unchanged stage. Application checksum changes are not Drive revision counts. After recording results, use the editor's confirmed Move to trash to clean up the synthetic entry.

## Gates that remain after a successful smoke test

- A checksum read followed by upload is not atomic against external Drive writers. Same-process locks serialize app writes only. If a metadata ETag is supplied, the adapter forwards If-Match; real Drive ETag/precondition behavior is unverified. No complete external/multi-instance lost-update protection is claimed. Production enabling requires a verified concurrency policy.
- Repeat reliability/conflict tests and record actual Drive revision and cold/warm latency measurements. No performance or revision-retention guarantee follows from mocked tests.
- Production needs HTTPS/Secure cookies, Console publishing/domain/homepage/privacy checks, remaining UI/release acceptance, dependency/privacy audit, and explicit production enablement. External apps in Testing may have seven-day refresh-token limits despite the app's 30-day session.

References used for implementation: [Google web-server OAuth](https://developers.google.com/identity/protocols/oauth2/web-server), [Google OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect), and [Drive revision management](https://developers.google.com/workspace/drive/api/guides/manage-revisions). The installed Go OAuth and ID-token library sources were inspected; no new dependency was added. No authoritative Drive v3 conditional-upload guarantee was established, so optional If-Match forwarding is not presented as an atomicity guarantee.
