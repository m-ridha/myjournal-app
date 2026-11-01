// spec: specs/myjournal-e2e.md
// seed: tests/e2e/seed.spec.ts

import { test, expect, type Locator, type Page } from "@playwright/test";

const catalogs: Record<string, Record<string, string>> = {
  en: require("../../assets/locales/en.json"),
  id: require("../../assets/locales/id.json"),
};

async function fitsViewport(page: Page, element?: Locator) {
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  if (element) {
    const bounds = await element.boundingBox();
    expect(bounds).not.toBeNull();
    expect(bounds!.x).toBeGreaterThanOrEqual(0);
    expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(
      page.viewportSize()!.width + 1,
    );
  }
}

async function alignedMobileHeader(page: Page) {
  if (page.viewportSize()!.width > 640) {
    return;
  }
  const banner = page.getByRole("banner");
  const edges = await banner.locator(".mj-header-inner").evaluate(element => {
    const bounds = element.getBoundingClientRect();
    const style = getComputedStyle(element);
    return {
      left: bounds.left + Number.parseFloat(style.paddingLeft),
      right: bounds.right - Number.parseFloat(style.paddingRight),
      center: (bounds.left + bounds.right) / 2,
    };
  });
  const brand = await banner.locator(".mj-brand").boundingBox();
  const actions = await banner.locator(".mj-header-actions").boundingBox();
  expect(Math.abs(brand!.x - edges.left)).toBeLessThanOrEqual(1);
  expect(
    Math.abs(actions!.x + actions!.width - edges.right),
  ).toBeLessThanOrEqual(1);
  const links = banner.locator(".mj-nav a");
  if (await links.count()) {
    const first = await links.first().boundingBox();
    const last = await links.last().boundingBox();
    const navigationCenter = (first!.x + last!.x + last!.width) / 2;
    expect(Math.abs(navigationCenter - edges.center)).toBeLessThanOrEqual(1);
  }
}

for (const locale of ["en", "id"]) {
  for (const theme of ["light", "dark"]) {
    for (const width of [320, 390, 640, 1440]) {
      test(`${locale} ${theme} ${width}px whole-app presentation`, async ({
        page,
        context,
      }, testInfo) => {
        test.setTimeout(90_000);
        const messages = catalogs[locale];
        const pageErrors: string[] = [];
        const externalRequests: string[] = [];
        page.on("pageerror", (error) => pageErrors.push(error.message));
        page.on("request", (request) => {
          if (
            new URL(request.url()).origin !==
            new URL(testInfo.project.use.baseURL as string).origin
          ) {
            externalRequests.push(request.url());
          }
        });

        // 1. Open the localized landing page with the chosen theme and viewport; expand both guide sections.
        await page.setViewportSize({ width, height: 900 });
        await context.addCookies([
          {
            name: "mj_language",
            value: locale,
            url: testInfo.project.use.baseURL as string,
          },
        ]);
        await page.addInitScript(
          (selectedTheme) => localStorage.setItem("mj-theme", selectedTheme),
          theme,
        );
        await page.goto(`/${locale}/`);
        await expect(page.locator("html")).toHaveAttribute("lang", locale);
        await expect(page.locator("html")).toHaveAttribute("data-theme", theme);
        await expect(
          page.getByRole("heading", {
            name: messages["landing.title"],
            level: 1,
          }),
        ).toBeVisible();
        await page
          .getByText(messages["landing.start.title"], { exact: true })
          .click();
        await page
          .getByText(messages["landing.questions.title"], { exact: true })
          .click();
        await expect(
          page.getByText(messages["landing.questions.offline.body"], {
            exact: true,
          }),
        ).toBeVisible();
        await fitsViewport(page);
        await alignedMobileHeader(page);
        await page.evaluate(() => window.scrollTo(0, 0));
        await page.screenshot({
          path: testInfo.outputPath("landing-expanded.png"),
          fullPage: true,
        });

        // 2. Open the public language menu with the keyboard; verify its bounds and Escape focus return.
        const publicLanguage = page
          .getByLabel(messages["nav.changeLanguage"], { exact: true })
          .and(page.locator("summary"));
        await publicLanguage.focus();
        await page.keyboard.press("Enter");
        const publicMenu = page.getByRole("navigation", {
          name: messages["nav.changeLanguage"],
        });
        await expect(publicMenu).toBeVisible();
        await fitsViewport(page, publicMenu);
        await page.keyboard.press("Escape");
        await expect(publicMenu).toBeHidden();
        await expect(publicLanguage).toBeFocused();

        // 3. Check public reading pages, the sticky Help anchor, sign-in/failure, signed-out Settings and 404.
        const publicPaths = [
          `/${locale}/help#drive-access`,
          `/${locale}/privacy`,
          `/${locale}/terms`,
          "/sign-in",
          "/sign-in-failed",
          "/settings",
          "/missing-redesign-page",
        ];
        for (const path of publicPaths) {
          await page.goto(path);
          await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
          await fitsViewport(page);
          if (path.includes("#drive-access")) {
            const heading = page.getByRole("heading", {
              name: messages["help.access.title"],
            });
            const anchorBounds = await heading.boundingBox();
            const headerBounds = await page.getByRole("banner").boundingBox();
            expect(anchorBounds!.y).toBeGreaterThanOrEqual(
              headerBounds!.y + headerBounds!.height,
            );
            await page.screenshot({
              path: testInfo.outputPath("help-anchor.png"),
            });
          }
        }

        // 4. Sign in locally; inspect editor, Entries, Calendar and Settings without changing saved fixtures.
        await page.goto("/api/v1/auth/local/login");
        const editor = page.getByRole("textbox", {
          name: messages["editor.label"],
        });
        await expect(editor).toBeEditable();
        await expect(
          page.getByRole("button", { name: messages["editor.nextDay"] }),
        ).toBeDisabled();
        await fitsViewport(page, editor);
        await alignedMobileHeader(page);
        const writingEdges = await editor.evaluate(element => {
          const field = element.parentElement!.getBoundingClientRect();
          const container = document.querySelector(".mj-header-inner")!;
          const bounds = container.getBoundingClientRect();
          const style = getComputedStyle(container);
          return {
            left: field.left,
            right: field.right,
            expectedLeft: bounds.left + Number.parseFloat(style.paddingLeft),
            expectedRight: bounds.right - Number.parseFloat(style.paddingRight),
          };
        });
        expect(
          Math.abs(writingEdges.left - writingEdges.expectedLeft),
        ).toBeLessThanOrEqual(1);
        expect(
          Math.abs(writingEdges.right - writingEdges.expectedRight),
        ).toBeLessThanOrEqual(1);
        await page.goto("/sign-in-failed");
        await page
          .getByRole("link", {
            name: messages["signInFailed.tryAgain"],
            exact: true,
          })
          .click();
        await expect(page).toHaveURL(/\/sign-in$/);
        await expect(
          page.getByRole("link", {
            name: messages["landing.connectLocal"],
            exact: true,
          }),
        ).toBeVisible();
        await page.goto("/");
        await expect(editor).toBeEditable();
        await page.goto("/entries");
        await expect(
          page.getByRole("heading", {
            name: messages["entries.title"],
            level: 1,
          }),
        ).toBeVisible();
        await expect(
          page.getByText(messages["entries.loading"], { exact: true }),
        ).toBeHidden();
        const entryList = page.locator("[data-entry-list]");
        await expect(entryList).toHaveAttribute("aria-busy", "false");
        const entryRows = entryList.getByRole("link");
        const emptyEntries = entryList.getByText(messages["entries.empty"], {
          exact: true,
        });
        await expect(entryRows.or(emptyEntries).first()).toBeVisible();
        await expect(
          page.getByText(messages["entries.unavailable"], { exact: true }),
        ).toHaveCount(0);
        await fitsViewport(page);
        await page.goto("/calendar");
        await expect(
          page.getByRole("group", { name: messages["calendar.label"] }),
        ).toHaveAttribute("aria-busy", "false");
        await expect(
          page.getByRole("button", { name: messages["calendar.nextMonth"] }),
        ).toBeDisabled();

        // 5. Expand the month picker using the keyboard; verify future bounds, separated days and focus restoration.
        const monthTrigger = page.getByRole("button", {
          name: new RegExp(`^${messages["calendar.chooseMonth"]}:`),
        });
        await monthTrigger.focus();
        await page.keyboard.press("Enter");
        const monthPicker = page.getByRole("dialog", {
          name: messages["calendar.chooseMonth"],
        });
        await expect(monthPicker).toBeVisible();
        await fitsViewport(page, monthPicker);
        await expect(
          page.getByRole("button", { name: messages["calendar.nextYear"] }),
        ).toBeDisabled();
        await page.screenshot({
          path: testInfo.outputPath("calendar-picker.png"),
          fullPage: true,
        });
        await page.keyboard.press("Escape");
        await expect(monthTrigger).toBeFocused();
        const dayButtons = page
          .getByRole("group", { name: messages["calendar.label"] })
          .getByRole("button");
        const firstDay = await dayButtons.nth(0).boundingBox();
        const secondDay = await dayButtons.nth(1).boundingBox();
        expect(firstDay!.x + firstDay!.width).toBeLessThanOrEqual(
          secondDay!.x + 1,
        );
        await fitsViewport(page);

        // 6. Inspect protected language menus and focus; a Settings choice must change locale only after explicit Save.
        await page.goto("/settings");
        const headerLanguage = page.getByRole("banner").getByRole("button", {
          name: messages["nav.changeLanguage"],
        });
        await headerLanguage.focus();
        await page.keyboard.press("Enter");
        const headerMenu = page.getByRole("banner").getByRole("listbox");
        await expect(headerMenu).toBeVisible();
        await fitsViewport(page, headerMenu);
        await page.keyboard.press("Escape");
        await expect(headerLanguage).toBeFocused();
        const settingsLanguage = page.getByRole("button", {
          name: new RegExp(`^${messages["settings.language"]} `),
        });
        await settingsLanguage.focus();
        await page.keyboard.press("ArrowDown");
        const settingsMenu = page.getByRole("listbox", {
          name: messages["settings.language"],
          exact: true,
        });
        await expect(settingsMenu).toBeVisible();
        await fitsViewport(page, settingsMenu);
        await page.evaluate(() => window.scrollTo(0, 0));
        await page.screenshot({
          path: testInfo.outputPath("settings-language.png"),
          fullPage: true,
        });
        await page.keyboard.press("Escape");
        await expect(settingsLanguage).toBeFocused();
        const focusOutlineWidth = await settingsLanguage.evaluate((element) =>
          Number.parseFloat(getComputedStyle(element).outlineWidth),
        );
        expect(focusOutlineWidth).toBeGreaterThan(0);
        await expect(
          page.getByRole("button", { name: messages["settings.languageSave"] }),
        ).toBeVisible();
        await expect(page.locator("html")).toHaveAttribute("lang", locale);
        await fitsViewport(page);
        await settingsLanguage.click();
        const otherLocale = locale === "en" ? "id" : "en";
        await settingsMenu
          .getByRole("option", {
            name: otherLocale === "id" ? "Bahasa Indonesia" : "English",
            exact: true,
          })
          .click();
        await expect(page.locator("html")).toHaveAttribute("lang", locale);
        await page
          .getByRole("button", { name: messages["settings.languageSave"] })
          .click();
        await expect(page.locator("html")).toHaveAttribute("lang", otherLocale);
        expect(pageErrors).toEqual([]);
        expect(externalRequests).toEqual([]);
      });
    }
  }
}
