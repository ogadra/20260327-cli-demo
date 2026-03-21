import { expect, test } from "@playwright/test";

test("renders root element", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("#root")).toBeAttached();
});

test("page has correct title", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveTitle("front");
});
