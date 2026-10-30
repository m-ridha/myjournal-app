// spec: specs/myjournal-e2e.md
// seed: tests/e2e/seed.spec.ts

import { test, expect } from "@playwright/test";

test.describe("Public entry and local sign in", () => {
  test("Landing offers local sign in and opens the editor", async ({
    page,
  }) => {
    // 1. Navigate to `/`.
    await page.goto("/");

    // 2. Verify the landing heading "Your personal journal, one day at a time." is visible.
    await expect(
      page.getByRole("heading", {
        name: "Your personal journal, one day at a time.",
      }),
    ).toBeVisible();

    // 3. Verify the header offers Settings and Help, and does not show Write, Entries or Calendar
    // navigation while signed out.
    const header = page.getByRole("banner");
    await expect(header.getByRole("link", { name: "Settings" })).toBeVisible();
    await expect(header.getByRole("link", { name: "Help" })).toBeVisible();
    await expect(
      header.getByRole("link", { name: "Write", exact: true }),
    ).toHaveCount(0);
    await expect(
      header.getByRole("link", { name: "Entries", exact: true }),
    ).toHaveCount(0);
    await expect(
      header.getByRole("link", { name: "Calendar", exact: true }),
    ).toHaveCount(0);

    // 4. Click "Continue with Local Dev".
    await page.getByRole("link", { name: "Continue with Local Dev" }).click();

    // 5. Verify the URL is `/entries/today` and the "Journal entry" textarea becomes editable.
    // The app resolves `/entries/today` to the concrete date URL, e.g. `/entries/2026-10-30`.
    await expect(page).toHaveURL(/\/entries\/(today|\d{4}-\d{2}-\d{2})$/);
    await expect(
      page.getByRole("textbox", { name: "Journal entry" }),
    ).toBeEditable();
  });
});
