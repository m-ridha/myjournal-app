// spec: specs/myjournal-e2e.md
// seed: tests/e2e/seed.spec.ts

import { test, expect } from "@playwright/test";

test.describe("Calendar navigation", () => {
  test("Calendar opens the month picker and selects a month", async ({
    page,
  }) => {
    const calendarRequests: string[] = [];
    page.on("request", (request) => {
      if (request.url().includes("/api/v1/calendar")) {
        calendarRequests.push(request.url());
      }
    });

    const previousYear = new Date().getFullYear() - 1;
    const firstMonthName = `January ${previousYear}`;

    const journalEntry = page.getByRole("textbox", { name: "Journal entry" });
    const monthTrigger = page.getByRole("button", {
      name: /^Choose month and year:/,
    });
    const monthPicker = page.getByRole("dialog", {
      name: "Choose month and year",
    });
    const calendarGrid = page.getByRole("group", { name: "Journal calendar" });

    // 1. Sign in by navigating to `/api/v1/auth/local/login` and wait for the editor to become editable.
    await page.goto("/api/v1/auth/local/login");
    await expect(journalEntry).toBeEditable();

    // 2. Open `/calendar` and wait for the grid to finish loading.
    await page.goto("/calendar");
    await expect(calendarGrid).toHaveAttribute("aria-busy", "false");

    // 3. Verify the next-month control is disabled while the current month is shown.
    await expect(
      page.getByRole("button", { name: "Next month" }),
    ).toBeDisabled();

    // 4. Open the month picker, verify it is visible, press Escape, and verify focus returns to the trigger.
    await monthTrigger.click();
    await expect(monthPicker).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(monthPicker).toBeHidden();
    await expect(monthTrigger).toBeFocused();

    // 5. Open the picker again, go to the previous year, select the first month option,
    // and verify the calendar label and grid update to the selected month.
    const requestsBeforeBrowsing = calendarRequests.length;
    await monthTrigger.click();
    await page.getByRole("button", { name: "Previous year" }).click();
    const requestsWhileBrowsingOptions = calendarRequests.length;
    await page.getByRole("button", { name: firstMonthName }).click();
    await expect(monthTrigger).toHaveAccessibleName(
      `Choose month and year: ${firstMonthName}`,
    );
    await expect(
      calendarGrid.getByRole("button", { name: `January 1, ${previousYear}` }),
    ).toBeVisible();

    // 6. Verify no calendar request is made while browsing picker options, before a month is chosen.
    expect(requestsWhileBrowsingOptions).toBe(requestsBeforeBrowsing);
    expect(calendarRequests.length).toBeGreaterThan(
      requestsWhileBrowsingOptions,
    );
  });
});
