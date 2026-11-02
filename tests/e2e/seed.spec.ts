// spec: specs/myjournal-e2e.md
// seed: tests/e2e/seed.spec.ts

import { test, expect } from "@playwright/test";

test("Local fixture is healthy", async ({ request }) => {
  // 1. Verify the isolated app is ready before exercising product journeys.
  const response = await request.get("/healthz");
  expect(response.ok()).toBe(true);
});
