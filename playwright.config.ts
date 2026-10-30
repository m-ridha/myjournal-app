import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:4173";
const port = new URL(baseURL).port || "80";
// Fixed test-only fixture for the passwordless local provider; never reuse outside e2e.
const sessionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

/**
 * End-to-end checks for the local-provider fixture. See specs/myjournal-e2e.md.
 * The suite starts its own app on the configured port with an isolated data directory.
 */
export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: "html",
  use: {
    baseURL,
    trace: "on-first-retry",
  },
  webServer: {
    command: [
      `PORT=${port}`,
      `APP_BASE_URL=${baseURL}`,
      "PROVIDERS=local",
      "COOKIE_SECURE=false",
      `SESSION_KEY=${sessionKey}`,
      "LOCAL_DATA_DIR=.devdata/e2e",
      "go run ./cmd/app",
    ].join(" "),
    url: `${baseURL}/healthz`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "firefox",
      use: { ...devices["Desktop Firefox"] },
    },
    {
      name: "webkit",
      use: { ...devices["Desktop Safari"] },
    },
  ],
});
