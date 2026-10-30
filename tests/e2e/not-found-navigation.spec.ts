// spec: specs/myjournal-e2e.md
// seed: tests/e2e/seed.spec.ts

import { test, expect } from "@playwright/test";

test.describe("Signed-in and signed-out 404", () => {
  test("404 keeps navigation for signed-in users only", async ({ page }) => {
    const journalEntry = page.getByRole("textbox", { name: "Journal entry" });
    const header = page.getByRole("banner");

    // 1. Sign in by navigating to `/api/v1/auth/local/login` and wait for the editor to become editable.
    await page.goto("/api/v1/auth/local/login");
    await expect(journalEntry).toBeEditable();

    // 2. Navigate to `/definitely-missing` and verify the response status is 404.
    const signedInResponse = await page.goto("/definitely-missing");
    expect(signedInResponse?.status()).toBe(404);

    // 3. Verify "Page not found" is shown and the header shows the signed-in
    // Write, Entries and Calendar navigation.
    await expect(
      page.getByRole("heading", { name: "Page not found" }),
    ).toBeVisible();
    await expect(
      header.getByRole("link", { name: "Write", exact: true }),
    ).toBeVisible();
    await expect(
      header.getByRole("link", { name: "Entries", exact: true }),
    ).toBeVisible();
    await expect(
      header.getByRole("link", { name: "Calendar", exact: true }),
    ).toBeVisible();

    // 4. Sign out from `/settings` and wait until the signed-out landing is shown.
    await page.goto("/settings");
    await page.getByRole("button", { name: "Sign out" }).click();
    await expect(
      page.getByRole("heading", {
        name: "Your personal journal, one day at a time.",
      }),
    ).toBeVisible();

    // 5. Navigate to `/definitely-missing` again and verify the response status is 404 and
    // "Page not found" is shown, but the header no longer shows Write, Entries or Calendar.
    const signedOutResponse = await page.goto("/definitely-missing");
    expect(signedOutResponse?.status()).toBe(404);
    await expect(
      page.getByRole("heading", { name: "Page not found" }),
    ).toBeVisible();
    await expect(
      header.getByRole("link", { name: "Write", exact: true }),
    ).toHaveCount(0);
    await expect(
      header.getByRole("link", { name: "Entries", exact: true }),
    ).toHaveCount(0);
    await expect(
      header.getByRole("link", { name: "Calendar", exact: true }),
    ).toHaveCount(0);
  });
});
