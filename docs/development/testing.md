# Testing and acceptance

Use synthetic accounts/text and deployments you control. Never attach cookies, tokens, callback queries, account claims or journal content to reports. This guide owns repeatable checks; [status](status.md) records accepted evidence and coverage limits. Do not routinely repeat already accepted human Google experiments.

## Automated checks

Follow [README prerequisites](../../README.md#prerequisites) and pinned tools before running:

```sh
make fmt lint test
make test-architecture
python3 -B -m unittest discover -s scripts/python -p 'test_*.py'
```

Run architecture probes separately from builds/lint because they temporarily introduce forbidden imports and must clean up. `make test` runs seven JavaScript suites and Go tests. `make test-js` uses Node.js or macOS JavaScriptCore without npm packages; retain JSC success-marker checking so asynchronous failures cannot appear successful. Release helper tests also have `make test-release-script`. Python is needed for helper tests, not ordinary local runtime.

For documentation-only changes, verify links/paths, consistency and `git diff --check`; runtime tests are unnecessary. For implementation changes, run formatting, lint and tests. After relevant generator inputs change, regenerate OpenAPI, Wire or Mockery output with pinned tools and inspect diffs. Never hand-edit generated output. Go tests use fixed clocks, temporary directories, shared Store contracts, Testify and generated Mockery mocks; no live credentials in automated tests.

Optional container validation uses `docker build -t myjournal-local .` with a running daemon. The existing browser benchmark harness can measure synthetic interactions; retain new measurements outside version control and record setup/limits before drawing conclusions. No checked-in historical result bundle is required for setup or CI. For the [browser harness](../../scripts/javascript/benchmark-browser.js), sign in to Google on localhost, open `/privacy` with other editors idle, paste the script in DevTools and call `await mjBenchmark('YOUR-UNUSED-PAST-DATE')`. It writes synthetic data and trashes its fixture on success; after failure inspect the returned result before retrying or cleaning up.

## Editor and recovery regressions

1. With local storage and a past date, check empty/read-only bootstrap, Save disabled until GET, failed GET Retry, Unicode/line endings, long text and no future dates. Verify explicit dated URLs survive reload and timezone correction does not loop when storage is blocked.
2. Type during a delayed PUT. Confirm only acknowledged content becomes clean and later typing saves against its returned version. Check five-second debounce, unchanged suppression, one in-flight write and best-effort blur/hidden flushing.
3. Fail saves/reads with offline mode and synthetic 5xx responses. Text stays visible, unacknowledged backups remain, feedback is truthful and autosave pauses. Manual retry reads fresh first; test exact-match acknowledgement, differing content, failed verification and typing during retry.
4. Reload/navigate in the same tab with a draft; verify account/date isolation and explicit review of recovered/legacy drafts. Test empty edits, unavailable sessionStorage, migration failure and successful migration removal. Closed-tab expiry is accepted.
5. Save competing edits in two tabs. Exercise Keep mine, cancelled/confirmed Load the other version and Keep both, including another intervening edit. Keep both must wait for review before autosave. Ordinary cached reads and explicit fresh reconciliation must remain distinct.
6. Confirm trash, delay/fail DELETE and attempt blur/edit actions. Preserve text on failure; clear only this tab/date backup after 204. New saves must not overwrite a still-pending trash action.
7. Expire/revoke authentication: show the new-tab sign-in banner without navigation or draft loss. Reconnect to the same account and explicitly retry. A different-account reconnect must reject old-tab writes/disconnect before service calls. Check stale successes do not dismiss current expiry feedback.
8. Delay/fail logout and disconnect. Preserve session/drafts until acknowledgement; cleanup-only retry must not resend logout/revocation. Preserve other accounts/theme; verify cross-tab notification limits and local files after local disconnect.

## Browsing, language and accessibility

- Verify latest-10 Entries, fixed page-1 Retry, first meaningful preview lines and Calendar for older notes. API pagination rejects out-of-range values before provider access.
- Browse month/year picker grids without data fetches; check future/year-one bounds, keyboard focus/dismissal, localization and latest-request-only success/error rendering under reordered responses.
- Check English/Indonesian selection before/after sign-in and persistence after sign-out. Exercise header theme switching, device-theme changes without explicit preference and narrow-screen wrapping. Appearance controls stay out of Settings.
- At desktop and narrow device-toolbar widths, test keyboard-only navigation, visible focus, confirmation focus return, touch targets, reduced motion, native 200% zoom, text/non-text contrast and non-color status cues. Use a screen reader to inspect save/error/auth announcements and picker controls. Record browser/OS and manual versus automated coverage.
- Optional axe-core scans run only as local QA tooling, never as a runtime/CDN dependency. Review incomplete/manual results; a passing scan is not WCAG certification. Keep reports and synthetic fixture output outside version control. The [accessibility helper](../../scripts/python/check-accessibility.py) needs Python Playwright, its Chromium browser, a locally downloaded axe-core script and a built app binary; invoke it with `--app-binary`, `--axe-script` and `--report` paths. It starts an isolated local-provider fixture.

## Google integration

Use the private localhost configuration in [operations](operations.md), an authorized test user and an unused past date. Never broaden drive.file scope or publish the Console app for a smoke test.

1. Sign in with consent. Inspect cookie attributes/size without copying values: encrypted session HttpOnly, SameSite=Lax, Path=/ and fixed lifetime; pending login cookie removed after completion. Cancelled/partial consent must preserve an existing session and draft.
2. Save synthetic Unicode/multiline text; inspect exact plain-text Drive content/path. On a fresh account, reads/lists/calendar/settings must not create folders. Rename tagged folders and verify discovery; missing folders are recreated only on write. Preserve duplicate/data-bearing folders.
3. Restart with the same SESSION_KEY and read again to exercise token refresh. Reconnect without forced consent when available; omitted refresh-token preservation is only valid for the same account. Do not claim the omission occurred unless observed.
4. Exercise stale-tab conflicts, direct Drive edits/trash, and repeated trash/new-file preservation using synthetic data. These checks do not prove compare/upload atomicity against concurrent external writers.
5. Revoke the Google connection, restart to remove cached access tokens, and retry from an editor with unsaved text. Recover through the same-account new-tab sign-in flow. Check Settings Disconnect/reconnect separately; ordinary Sign out does not revoke access.
6. Inspect query-free logs and public HTTPS/cookie behavior. Record measured cold/warm timings and actual Drive Manage versions observations only if required; checksum changes/save counts are not retained revision counts.

## Browser security and release smoke

Inspect CSP/security headers and browser console/network under real navigation, save/conflict/recovery, language/theme, settings and calendar interactions. Confirm local external scripts load without CSP violations and no third-party page requests occur. Check HTTPS host-only HSTS, no caching of protected responses, unsafe cross-origin rejection, and callback errors without exposed query data. Distinguish console inspection from complete browser/security coverage.

Use [operations](operations.md) for authorized release/rollback checks. Automated release script fixtures test target/plan restrictions and deployment polling; they do not deploy. Record actual deployment identity and health separately from mocked success.
