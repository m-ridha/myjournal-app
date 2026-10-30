# myjournal end-to-end plan

**Seed:** `tests/e2e/seed.spec.ts`
**Base URL:** `http://localhost:4173` (local provider, isolated data directory `.devdata/e2e`)
**Specs:** one file per scenario in `tests/e2e/`; every test header names this plan and the seed.

## 1. Public entry and local sign in

### 1.1 Landing offers local sign in and opens the editor

**Spec:** `tests/e2e/landing-local-sign-in.spec.ts`

**Steps:**

1. Navigate to `/`.
2. Verify the landing heading "Your personal journal, one day at a time." is visible.
3. Verify the header offers Settings and Help, and does not show Write, Entries or Calendar navigation while signed out.
4. Click "Continue with Local Dev".
5. Verify the URL resolves to `/entries/YYYY-MM-DD` and the "Journal entry" textarea becomes editable.

## 2. Writing and browsing entries

### 2.1 Signed-in user saves a past entry and finds it in Entries

**Spec:** `tests/e2e/editor-save-entry.spec.ts` (fixture date `2001-02-03`)

**Steps:**

1. Sign in through `/api/v1/auth/local/login` and wait for the editor to load.
2. Open `/entries/2001-02-03`, wait for the textarea to become editable.
3. Fill the entry with synthetic text and click Save.
4. Verify the status shows "Saved".
5. Reload the page and verify the same text is shown.
6. Open `/entries` and verify the saved date appears as a row linking to `/entries/2001-02-03`.

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
