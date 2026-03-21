import { expect, test } from "@playwright/test";

test("renders root element", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("#root")).toBeAttached();
});
