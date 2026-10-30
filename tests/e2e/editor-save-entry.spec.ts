// spec: specs/myjournal-e2e.md
// seed: tests/e2e/seed.spec.ts

import { test, expect } from "@playwright/test";

test.describe("Writing and browsing entries", () => {
  test("Signed-in user saves a past entry and finds it in Entries", async ({
    page,
  }) => {
    const entryDate = "2001-02-03";
    const entryText = "Synthetic e2e entry for 2001-02-03";
    const journalEntry = page.getByRole("textbox", { name: "Journal entry" });

    // 1. Sign in by navigating to `/api/v1/auth/local/login` and wait for the editor to load.
    // The app resolves `/entries/today` to the concrete date URL, e.g. `/entries/2026-10-30`.
    await page.goto("/api/v1/auth/local/login");
    await expect(page).toHaveURL(/\/entries\/(today|\d{4}-\d{2}-\d{2})$/);
    await expect(journalEntry).toBeEditable();

    // 2. Open `/entries/${entryDate}` and wait for the "Journal entry" textarea to become editable.
    await page.goto(`/entries/${entryDate}`);
    await expect(journalEntry).toBeEditable();

    // 3. Fill the entry with synthetic text and click Save.
    await journalEntry.fill(entryText);
    await page.getByRole("button", { name: "Save" }).click();

    // 4. Verify the status shows "Saved".
    await expect(page.getByRole("status")).toContainText("Saved");

    // 5. Reload the page and verify the same text is shown.
    await page.reload();
    await expect(journalEntry).toHaveValue(entryText);

    // 6. Open `/entries` and verify the saved date appears as a row linking to `/entries/${entryDate}`.
    await page.goto("/entries");
    const savedEntryRow = page.getByRole("link").filter({ hasText: entryText });
    await expect(savedEntryRow).toHaveAttribute(
      "href",
      `/entries/${entryDate}`,
    );
  });
});
