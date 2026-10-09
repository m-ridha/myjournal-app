# CSP and security-header browser QA — 2026-10-09

Focused verdict: **Ready with known limitations** for the CSP baseline in local development. No application defect was reproduced in the covered flows. Full Safari startup-console capture remains unverified; the broader Phase 2/3 and production release gates remain open.

## Environment and method

Tested the user's running app at `http://localhost:3030` with its uncommitted HTTP/CSP changes, using the Web App QA & Release Copilot workflow. No application code or server configuration was changed by this QA run.

- Google Chrome **155.0.8059.40**, installed browser driven headlessly with Python Playwright, isolated profile, 1280×900 viewport, Asia/Jakarta time zone.
- Native Safari **26.4 (21624.1.16.11.4)** driven through Apple's safaridriver/WebDriver. Initial session creation timed out; after the user confirmed remote automation was enabled, restarting only the QA-owned driver allowed sessions to start normally.
- Temporary automation scripts/dependencies/evidence were kept under `/tmp`, without adding repository dependencies. The development server was left running.
- Tested only unused past dates confirmed missing by the API. Synthetic entries were moved to trash after testing; their recoverable trash copies remain. No existing entry was edited or deleted. Captured screenshots contain only the header, excluding journal/account content. Cookies, account claims, and journal bodies were not logged.

## Automated browser evidence

| Journey/check | Chrome | Native Safari | Evidence |
| --- | --- | --- | --- |
| Landing and local login → loaded editor | Passed | Passed | Browser navigation and editable loaded textarea |
| Dynamic response headers | Passed | Passed | CSP, no-store, nosniff, DENY framing, no-referrer |
| Light/dark preference after reload | Passed | Passed | DOM theme state; Chrome also tested both choices separately |
| External theme bootstrap before CSS | Passed | Passed | Template order/resource availability; no inline-script exception |
| Header Settings gear | Passed | Passed | Visible element, same-origin CSS mask, inspected header screenshot |
| Synthetic plain-text/Unicode save | Passed | Passed | PUT succeeded, Saved status, acknowledged draft cleared |
| Saved list preview | Passed | Passed | Synthetic entry link and meaningful preview rendered |
| Calendar previous month → Today | Passed | Passed | Month changed and returned to original month |
| Unsaved draft → reload | Passed | Passed | Synthetic draft restored without replacing it with server text |
| Concurrent change → conflict → Keep mine | Passed | Passed | Simulated second writer; conflict controls displayed; conditional resolution saved local text |
| Settings → sign out | Passed | Passed | Pending tab backup cleared after logout; theme retained |
| Signed-out API request | Passed | Passed | 401 with security headers |
| Styled missing page | Passed | Passed | 404 page and security headers |
| Deliberately inserted inline script | Blocked as expected | Blocked as expected | Script did not execute; script-src-elem violation observed |
| Unavailable browser backup | Passed, simulated | Not covered | Chrome injected storage-access exceptions while retaining cookies; landing/editor/theme controls worked and backup warning was truthful |

Chrome's successful final flow produced **21 passing checks**, its separate theme/storage flow **6 passing checks**, and Safari's final flow **25 passing checks**. These are browser-harness assertions, not an additional full unit/lint/race run.

## Console, limitations, and harness corrections

Chrome captured no page exceptions or unexpected app CSP violations. Its console contained expected messages for the deliberately triggered HTTP 409 conflict, HTTP 404 missing page, and blocked inline-script probe. Those intentional negatives are not product defects.

Safari recorded no app CSP violations or script errors during the instrumented actions and blocked the inline probe. Its WebDriver interface did not supply a complete startup console history; event listeners installed after navigation cannot prove that history was clean. A manual Safari Web Inspector console check can close this narrow gap.

Two initial Chrome harness assertions needed correction: the preview is a sibling of the Open link rather than its text, and Playwright string-based wait evaluation encountered the app's unsafe-eval restriction. The final run used the correct preview container and function-based waits, without disabling or weakening CSP. These were automation issues; the app was unchanged. Cleanup ran for interrupted synthetic-entry tests as well.

The storage test simulated JavaScript storage failures; it was not Chrome's site-data-blocking setting, which can also prevent authentication cookies. Native Safari blocked-storage behavior, physical devices, full accessibility/visual regression, production HTTPS/Google OAuth, and every conflict-resolution variant were outside this focused run. Existing wider acceptance evidence remains separate.

Local evidence: `/tmp/myjournal-csp-qa-evidence/{chrome.json,chrome-storage.json,safari.json,chrome-header-theme.png,safari-header-theme.png}`. These temporary files may expire; this report preserves the scope and observations.
