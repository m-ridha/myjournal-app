// spec: specs/myjournal-e2e.md
// seed: tests/e2e/seed.spec.ts

import { test, expect } from "@playwright/test";

test.describe("Theme and language preferences", () => {
  test("Theme toggle and language choice persist across navigation", async ({
    page,
  }) => {
    const documentElement = page.locator("html");
    const languageControl = page.getByRole("banner").locator("summary");

    // 1. Open `/help`.
    await page.goto("/help");
    await expect(
      page.getByRole("heading", { name: "Help", level: 1 }),
    ).toBeVisible();

    // 2. Toggle the header theme control and verify `data-theme` changes on the document element.
    await expect(documentElement).toHaveAttribute("data-theme", "light");
    await page.getByRole("button", { name: "Switch to dark mode" }).click();
    await expect(documentElement).toHaveAttribute("data-theme", "dark");

    // 3. Navigate to `/privacy` and verify the chosen theme is still applied.
    await page.goto("/privacy");
    await expect(documentElement).toHaveAttribute("data-theme", "dark");

    // 4. Use the public header language control to choose Bahasa Indonesia and verify the
    // document element switches to `lang="id"`. The control keeps the current page, so
    // choosing Indonesian from `/privacy` lands on `/id/privacy` (`/id/help` when started
    // from `/help`).
    await languageControl.click();
    await page.getByRole("link", { name: "Bahasa Indonesia" }).click();
    await expect(page).toHaveURL(/\/id\/privacy$/);
    await expect(documentElement).toHaveAttribute("lang", "id");

    // 5. Navigate to `/id/privacy` and verify English is still available from the language control.
    await page.goto("/id/privacy");
    await languageControl.click();
    await expect(
      page
        .getByRole("banner")
        .getByRole("link", { name: "English", exact: true }),
    ).toBeVisible();
  });
});
