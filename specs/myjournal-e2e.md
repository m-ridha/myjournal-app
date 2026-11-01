# myjournal end-to-end plan

**Seed:** `tests/e2e/seed.spec.ts`
**Base URL:** `http://localhost:4173` (local provider, isolated data directory `.devdata/e2e`)
**Execution:** Start and stop an owned local-provider server; an occupied port must fail startup without running tests against or stopping the existing listener. Run suites sequentially because fixture data is shared. Retain traces and screenshots for failures.
**Specs:** one file per scenario in `tests/e2e/`; every test header names this plan and the seed.

## 0. Local fixture health

**Spec:** `tests/e2e/seed.spec.ts`

1. Request `/healthz` and verify the isolated local app responds successfully.

## 1. Public entry and local sign in

### 1.1 Landing offers local sign in and opens the editor

**Spec:** `tests/e2e/landing-local-sign-in.spec.ts`

**Steps:**

1. Navigate to `/`.
2. Verify the landing heading "Your personal journal, one day at a time." is visible.
3. Verify the header offers Settings and Help, and does not show Write, Entries or Calendar navigation while signed out.
4. Click the localized local-development sign-in action.
5. Verify the URL resolves to `/entries/YYYY-MM-DD` and the "Journal entry" textarea becomes editable.

## 2. Writing and browsing entries

### 2.1 Signed-in user saves a past entry and finds it in Entries

**Spec:** `tests/e2e/editor-save-entry.spec.ts` (engine-isolated fixture dates `2001-02-03` through `2001-02-05`)

**Steps:**

1. Sign in through `/api/v1/auth/local/login` and wait for the editor to load.
2. Open `/entries/{fixtureDate}`, wait for the textarea to become editable.
3. Fill the entry with synthetic text and click Save.
4. Verify the status shows "Saved".
5. Reload the page and verify the same text is shown.
6. Open `/entries` and verify the saved date appears as a row linking to `/entries/{fixtureDate}`.

## 3. Calendar navigation

### 3.1 Calendar opens the month picker and selects a month

**Spec:** `tests/e2e/calendar-month-picker.spec.ts`

**Steps:**

1. Sign in through `/api/v1/auth/local/login`.
2. Open `/calendar` and wait for the grid to finish loading.
3. Verify the next-month control is disabled while the current month is shown.
4. Open the month picker, verify it is visible, press Escape, and verify focus returns to the trigger.
5. Open the picker again, choose the previous year, select the first month, and verify the calendar label and grid update.
6. Verify no calendar request is made while browsing picker options before a month is chosen.

## 4. Signed-in and signed-out 404

### 4.1 404 keeps navigation for signed-in users only

**Spec:** `tests/e2e/not-found-navigation.spec.ts`

**Steps:**

1. Sign in through `/api/v1/auth/local/login`.
2. Navigate to `/definitely-missing`.
3. Verify the response status is 404 and "Page not found" is shown with the Write, Entries and Calendar navigation.
4. Sign out from `/settings`.
5. Navigate to `/definitely-missing` again and verify "Page not found" is shown without signed-in navigation.

## 5. Theme and language preferences

### 5.1 Theme toggle and language choice persist across navigation

**Spec:** `tests/e2e/theme-language-preference.spec.ts`

**Steps:**

1. Open `/help`.
2. Toggle the header theme control and verify `data-theme` changes on the document element.
3. Navigate to `/privacy` and verify the chosen theme persists.
4. Use the public header language picker to choose Bahasa Indonesia and verify the URL becomes `/id/privacy` (the control keeps the current page) with `lang="id"`.
5. Navigate to `/id/privacy` and verify English is still available from the language picker.

## 6. Whole-app responsive presentation

### 6.1 EN/ID and light/dark pages remain usable at 320px, 390px, 640px and 1440px

**Spec:** `tests/e2e/redesign-presentation.spec.ts` (eight combinations per browser)

**Steps:**

1. Open the localized landing page at the chosen viewport/theme and expand the journaling guide and questions. Verify localized content, theme and no horizontal overflow.
2. Open the public language menu with the keyboard. Verify the popup stays within the viewport and Escape restores focus.
3. Visit Help at its Drive-access anchor, Privacy, Terms, sign-in, sign-in failure, signed-out Settings and 404. Verify visible main headings and contained content; the Help target must remain below the sticky header.
4. Sign in locally and inspect the editor, Entries, Calendar and Settings. Verify editable writing, disabled future navigation, completed loading and contained content. Entries must show loaded entry links or its localized empty message; unavailable feedback is a failure. With an existing local session, follow the sign-in-failure retry link to `/sign-in`, verify a login action is offered, then return home and verify the existing session still opens an editable entry. At mobile widths verify left-aligned brand, right-aligned action controls and centered page-navigation links. Verify the writing field aligns with the shared header content edges, including wide screens.
5. Open the month picker with the keyboard. Verify popup bounds, the disabled future-year control, non-overlapping day cells and Escape focus return.
6. Open protected header and Settings language menus, check their bounds and Escape focus return with a visible Settings focus outline. Choose another Settings language and verify the locale changes only after Save language. Collect screenshots, uncaught page errors and off-origin page requests.

## 7. Editor and cleanup recovery presentation

### 7.1 Long text and recovery controls remain usable across locales and themes

**Spec:** `tests/e2e/editor-recovery-layout.spec.ts` (EN/light desktop, EN/dark narrow, ID/light narrow, ID/dark desktop per browser; independent engine/case fixture dates in January 1990)

**Steps:**

1. Sign in to the real local provider. With a separately authenticated API context, delete this synthetic fixture if present and verify a fresh read acknowledges absence before installing any failure routes. Return one synthetic 503 for the entry read, and verify disabled writing/Save, truthful failure feedback and working Retry. Once the empty editor is editable and unfocused, verify its computed boundary contrast is at least 3:1 against both adjacent backgrounds.
2. Fill long synthetic text and return one synthetic 503 from Save. Verify retained editable text, contained 44px controls and successful manual retry against the real server.
3. Reload the saved long entry and verify the text remains editable and exact. Scroll the textarea to 25% and 70% of its internal scroll range and verify both are interior positions. At each position require the visible, enabled trash control's entire hit area to remain outside the textarea writing viewport, focus it and capture a screenshot. Cover narrow and desktop widths in both themes and languages.
4. Return one synthetic 401 from the next Save. Verify the authentication banner, same URL, retained editable text and separate-tab sign-in link. Retry against the still-authenticated local server and verify the banner clears after acknowledgement.
5. Write a competing version through the real local API, submit this editor's stale version and verify the actual conflict panel and all three enabled choices. Choose Keep both, review retained local and server text, and explicitly Save.
6. Confirm trash and verify the real 204 acknowledgement clears the editor.
7. Cause browser draft removal to fail after a real acknowledged logout. Verify truthful cleanup failure, disabled Disconnect and a working cleanup retry that does not issue another logout. Capture each recovery state and uncaught page errors.

Close the scenario page and require a fresh absent read in teardown, including after failure; delete any present fixture only with a 204 acknowledgement. The independent API context retains authentication through the UI logout. A forcefully interrupted process may skip teardown; the next run still resets its fixed synthetic date before relying on emptiness. Unexpected cleanup statuses fail the check. Keep the confirmed UI trash step as separate journey coverage.

Synthetic 401/503 responses and browser-storage failures exercise the production DOM and scripts; they do not establish real Google expiry/reconnection behavior. Conflicts, successful saves, trash and logout use the isolated local provider. Existing JavaScript suites retain the detailed concurrency, account isolation and ambiguous-acknowledgement checks. Screenshots are review evidence rather than pixel-diff or accessibility certification.
