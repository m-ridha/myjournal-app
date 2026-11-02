// spec: specs/myjournal-e2e.md
// seed: tests/e2e/seed.spec.ts

import {
  test,
  expect,
  type APIRequestContext,
  type Page,
  type TestInfo,
} from "@playwright/test";

const catalogs: Record<string, Record<string, string>> = {
  en: require("../../assets/locales/en.json"),
  id: require("../../assets/locales/id.json"),
};
const cases = [
  { locale: "en", theme: "light", width: 1440 },
  { locale: "en", theme: "dark", width: 320 },
  { locale: "id", theme: "light", width: 320 },
  { locale: "id", theme: "dark", width: 1440 },
];

async function clearFixture(request: APIRequestContext, entryURL: string) {
  const before = await request.get(entryURL, {
    headers: { "Cache-Control": "no-store" },
  });
  expect(
    before.status(),
    `Fresh fixture read before cleanup for ${entryURL}`,
  ).toBe(200);
  const previous = await before.json();
  expect(typeof previous.exists).toBe("boolean");
  if (previous.exists) {
    const deletion = await request.delete(entryURL);
    expect(deletion.status(), `Fixture deletion for ${entryURL}`).toBe(204);
  }
  const current = await request.get(entryURL, {
    headers: { "Cache-Control": "no-store" },
  });
  expect(
    current.status(),
    `Fresh fixture read after cleanup for ${entryURL}`,
  ).toBe(200);
  expect(await current.json()).toMatchObject({
    text: "",
    version: "",
    exists: false,
  });
}

async function recordLayout(page: Page, testInfo: TestInfo, name: string) {
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  for (const button of await page.getByRole("main").getByRole("button").all()) {
    if (!(await button.isVisible())) {
      continue;
    }
    const bounds = await button.boundingBox();
    expect(bounds!.x).toBeGreaterThanOrEqual(0);
    expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(
      page.viewportSize()!.width + 1,
    );
    // Firefox can report a nominal 44px control a fraction of a pixel below 44.
    expect(bounds!.height).toBeGreaterThanOrEqual(43.99);
  }
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({
    path: testInfo.outputPath(`${name}.png`),
    fullPage: true,
  });
}

for (const [caseIndex, scenario] of cases.entries()) {
  test(`${scenario.locale} ${scenario.theme} ${scenario.width}px editor recovery layout`, async ({
    page,
    context,
    request,
  }, testInfo) => {
    test.setTimeout(90_000);
    const messages = catalogs[scenario.locale];
    const engineIndex = ["chromium", "firefox", "webkit"].indexOf(
      testInfo.project.name,
    );
    const entryDate = `1990-01-${String(engineIndex * cases.length + caseIndex + 1).padStart(2, "0")}`;
    const entryURL = `/api/v1/entries/${entryDate}`;
    const reflection = "Synthetic journal reflection — catatan pengujian.\n";
    const longLine = "Long unbroken line ".repeat(12);
    const text = (reflection + longLine + "\n").repeat(30);
    const editor = page.getByRole("textbox", {
      name: messages["editor.label"],
    });
    const save = page.getByRole("button", {
      name: messages["editor.save"],
      exact: true,
    });
    const status = page.getByRole("status");
    const pageErrors: string[] = [];
    page.on("pageerror", (error) => pageErrors.push(error.message));

    const fixtureLogin = await request.get("/api/v1/auth/local/login");
    expect(fixtureLogin.ok()).toBe(true);
    try {
      // 1. Reset the synthetic fixture, then fail its initial read; Retry must restore an empty editable editor.
      await page.setViewportSize({ width: scenario.width, height: 900 });
      await context.addCookies([
        {
          name: "mj_language",
          value: scenario.locale,
          url: testInfo.project.use.baseURL as string,
        },
      ]);
      await page.addInitScript(
        (theme) => localStorage.setItem("mj-theme", theme),
        scenario.theme,
      );
      await page.goto("/api/v1/auth/local/login");
      await expect(page.locator("html")).toHaveAttribute(
        "data-theme",
        scenario.theme,
      );
      await expect(editor).toBeEditable();
      await clearFixture(request, entryURL);
      await page.route(
        `**${entryURL}`,
        async (route) => {
          await route.fulfill({
            status: 503,
            json: { error: "Synthetic read failure" },
          });
        },
        { times: 1 },
      );
      await page.goto(`/entries/${entryDate}`);
      await expect(status).toContainText(messages["editor.loadFailed"]);
      await expect(editor).not.toBeEditable();
      await expect(save).toBeDisabled();
      await recordLayout(page, testInfo, "load-failed");
      await page
        .getByRole("button", { name: messages["editor.retryLoad"] })
        .click();
      await expect(editor).toBeEditable();
      await expect(editor).toHaveValue("");
      await expect(editor).not.toBeFocused();
      const boundaryContrast = await editor.evaluate((element) => {
        const field = element.parentElement!;
        const fieldStyle = getComputedStyle(field);
        const borderColor = fieldStyle.borderTopColor;
        const fieldColor = fieldStyle.backgroundColor;
        const surroundingColor = getComputedStyle(
          document.body,
        ).backgroundColor;
        function luminance(color: string) {
          const channels = color.match(/[\d.]+/g)!.map(Number);
          if (channels.length > 3 && channels[3] !== 1) {
            throw new Error("Editor boundary contrast requires opaque colors");
          }
          const linear = channels.slice(0, 3).map((channel) => {
            const value = channel / 255;
            return value <= 0.04045
              ? value / 12.92
              : ((value + 0.055) / 1.055) ** 2.4;
          });
          return linear[0] * 0.2126 + linear[1] * 0.7152 + linear[2] * 0.0722;
        }
        function ratio(background: string) {
          const border = luminance(borderColor);
          const adjacent = luminance(background);
          return (
            (Math.max(border, adjacent) + 0.05) /
            (Math.min(border, adjacent) + 0.05)
          );
        }
        return {
          borderColor,
          fieldColor,
          surroundingColor,
          borderWidth: Number.parseFloat(fieldStyle.borderTopWidth),
          innerRatio: ratio(fieldColor),
          outerRatio: ratio(surroundingColor),
        };
      });
      expect(boundaryContrast.borderWidth).toBeGreaterThan(0);
      expect(boundaryContrast.innerRatio).toBeGreaterThanOrEqual(3);
      expect(boundaryContrast.outerRatio).toBeGreaterThanOrEqual(3);
      await testInfo.attach("empty-editor-boundary-contrast", {
        body: JSON.stringify(boundaryContrast, null, 2),
        contentType: "application/json",
      });

      // 2. Fail a long-text save and retain the editable text; a manual retry must reach a real saved acknowledgement.
      await page.route(
        `**${entryURL}`,
        async (route) => {
          await route.fulfill({
            status: 503,
            json: { error: "Synthetic save failure" },
          });
        },
        { times: 1 },
      );
      await editor.fill(text);
      await save.click();
      await expect(status).toContainText(messages["editor.saveFailed"]);
      await expect(editor).toHaveValue(text);
      await expect(editor).toBeEditable();
      await recordLayout(page, testInfo, "save-failed");
      await save.click();
      await expect(status).toContainText(messages["editor.saved"]);

      // 3. Reload the saved long entry and scroll inside it; the trash hit area must stay outside the writing viewport.
      await page.reload();
      await expect(editor).toBeEditable();
      await expect(editor).toHaveValue(text);
      const trash = page.getByRole("button", {
        name: messages["editor.delete"],
        exact: true,
      });
      await expect(trash).toBeEnabled();
      for (const fraction of [0.25, 0.7]) {
        const scroll = await editor.evaluate((element, position) => {
          const maximum = element.scrollHeight - element.clientHeight;
          element.scrollTop = maximum * position;
          return { top: element.scrollTop, maximum };
        }, fraction);
        expect(scroll.top).toBeGreaterThan(0);
        expect(scroll.top).toBeLessThan(scroll.maximum);
        const writingBounds = await editor.boundingBox();
        const trashBounds = await trash.boundingBox();
        expect(writingBounds).not.toBeNull();
        expect(trashBounds).not.toBeNull();
        const intersectsWritingViewport =
          trashBounds!.x < writingBounds!.x + writingBounds!.width &&
          trashBounds!.x + trashBounds!.width > writingBounds!.x &&
          trashBounds!.y < writingBounds!.y + writingBounds!.height &&
          trashBounds!.y + trashBounds!.height > writingBounds!.y;
        expect(
          intersectsWritingViewport,
          "Trash hit area must not cover scrolling journal text",
        ).toBe(false);
        await trash.focus();
        await expect(trash).toBeFocused();
        await recordLayout(
          page,
          testInfo,
          `saved-entry-interior-scroll-${fraction * 100}`,
        );
      }

      // 4. Return a synthetic 401 from the next save; verify new-tab sign-in guidance without navigation or text loss, then retry.
      const revisedText = text + "Unsaved session-recovery text.";
      await page.route(
        `**${entryURL}`,
        async (route) => {
          await route.fulfill({
            status: 401,
            json: { error: "Synthetic expired session" },
          });
        },
        { times: 1 },
      );
      await editor.fill(revisedText);
      await save.click();
      const authPanel = page.getByRole("region", {
        name: messages["auth.title"],
      });
      await expect(authPanel).toBeVisible();
      await expect(
        authPanel.getByRole("link", { name: messages["auth.signInAgain"] }),
      ).toHaveAttribute("target", "_blank");
      await expect(page).toHaveURL(new RegExp(`/entries/${entryDate}$`));
      await expect(editor).toHaveValue(revisedText);
      await expect(editor).toBeEditable();
      await recordLayout(page, testInfo, "authentication-recovery");
      await save.click();
      await expect(authPanel).toBeHidden();
      await expect(status).toContainText(messages["editor.saved"]);

      // 5. Save a competing real API version, then submit this editor's stale version; Keep both must preserve both texts for review.
      const current = await page.request.get(entryURL, {
        headers: { "Cache-Control": "no-store" },
      });
      expect(current.ok()).toBe(true);
      const currentEntry = await current.json();
      const competingText = `Synthetic competing version ${testInfo.project.name} ${caseIndex}.`;
      const competingSave = await page.request.put(entryURL, {
        data: { text: competingText, baseVersion: currentEntry.version },
      });
      expect(competingSave.ok()).toBe(true);
      const localText =
        revisedText + "\nLocal changes retained during conflict.";
      await editor.fill(localText);
      await save.click();
      const conflict = page.getByRole("region", {
        name: messages["editor.conflictTitle"],
      });
      await expect(conflict).toBeVisible();
      await expect(editor).toHaveValue(localText);
      for (const key of [
        "editor.keepMine",
        "editor.loadOther",
        "editor.keepBoth",
      ]) {
        await expect(
          conflict.getByRole("button", { name: messages[key], exact: true }),
        ).toBeEnabled();
      }
      await recordLayout(page, testInfo, "real-version-conflict");
      await conflict
        .getByRole("button", { name: messages["editor.keepBoth"], exact: true })
        .click();
      await expect(conflict).toBeHidden();
      await expect(editor).toHaveValue(localText + "\n\n" + competingText);
      await expect(status).toContainText(messages["editor.unsaved"]);
      await save.click();
      await expect(status).toContainText(messages["editor.saved"]);

      // 6. Confirm trash only after saving reviewed text; the editor must clear after the real 204 acknowledgement.
      page.once("dialog", (dialog) => dialog.accept());
      await page
        .getByRole("button", { name: messages["editor.delete"], exact: true })
        .click();
      await expect(status).toContainText(messages["editor.deleted"]);
      await expect(editor).toHaveValue("");

      // 7. Fail browser draft removal after acknowledged logout; cleanup retry must stay local and never resend logout.
      await page.goto("/settings");
      await page.evaluate(() => {
        const scope = document.body.dataset.draftScope;
        sessionStorage.setItem(
          `mj-draft:v3:${scope}:1990-12-31`,
          "synthetic cleanup fixture",
        );
        const originalRemove = Storage.prototype.removeItem;
        (window as any).restoreDraftRemoval = () => {
          Storage.prototype.removeItem = originalRemove;
        };
        Storage.prototype.removeItem = function (key) {
          if (key.startsWith("mj-draft:v3:")) {
            throw new Error("Synthetic cleanup failure");
          }
          return originalRemove.call(this, key);
        };
      });
      let logoutRequests = 0;
      page.on("request", (request) => {
        if (new URL(request.url()).pathname === "/api/v1/logout") {
          logoutRequests += 1;
        }
      });
      await page
        .getByRole("button", {
          name: messages["settings.signOut"],
          exact: true,
        })
        .click();
      await expect(page.getByRole("status")).toContainText(
        messages["settings.signedOutCleanupFailed"],
      );
      await expect(
        page.getByRole("button", {
          name: messages["settings.disconnect"],
          exact: true,
        }),
      ).toBeDisabled();
      await recordLayout(page, testInfo, "logout-cleanup-failed");
      await page.evaluate(() => (window as any).restoreDraftRemoval());
      await page
        .getByRole("button", {
          name: messages["settings.retryDraftCleanup"],
          exact: true,
        })
        .click();
      await expect(
        page.getByRole("heading", {
          name: messages["landing.title"],
          level: 1,
        }),
      ).toBeVisible();
      expect(logoutRequests).toBe(1);
      expect(pageErrors).toEqual([]);
    } finally {
      try {
        if (!page.isClosed()) {
          await page.close();
        }
      } finally {
        await clearFixture(request, entryURL);
      }
    }
  });
}
