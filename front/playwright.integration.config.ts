import { defineConfig } from "@playwright/test";

/** Playwright configuration for integration tests against the full stack via compose. */
export default defineConfig({
  testDir: "./e2e",
  testMatch: "integration.spec.ts",
  timeout: 30_000,
  use: {
    baseURL: "http://localhost:5173",
  },
});
