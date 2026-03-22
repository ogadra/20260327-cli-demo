import { expect, test } from "@playwright/test";

/** Wait for the command input to be enabled, indicating the session is ready. */
async function waitForReady(page: import("@playwright/test").Page): Promise<void> {
  await expect(page.locator('input[placeholder="Enter command..."]')).toBeEnabled({
    timeout: 30_000,
  });
}

/** Execute a command and wait for the input to be re-enabled after completion. */
async function executeCommand(
  page: import("@playwright/test").Page,
  command: string,
): Promise<void> {
  const input = page.locator('input[placeholder="Enter command..."]');
  await input.fill(command);
  await input.press("Enter");
  await expect(input).toBeEnabled({ timeout: 30_000 });
}

/** Get the text content of the terminal output from xterm.js. */
async function getTerminalText(page: import("@playwright/test").Page): Promise<string> {
  return (await page.locator(".xterm-rows").textContent()) ?? "";
}

test.describe.serial("integration", () => {
  let sharedPage: import("@playwright/test").Page;

  test.beforeAll(async ({ browser }) => {
    sharedPage = await browser.newPage();
    await sharedPage.goto("http://localhost:5173");
    await waitForReady(sharedPage);
  });

  test.afterAll(async () => {
    await sharedPage.close();
  });

  test("executes command and shows output", async () => {
    await executeCommand(sharedPage, "pwd");
    const text = await getTerminalText(sharedPage);
    expect(text).toMatch(/\//);
  });

  test("shows prompt after completion", async () => {
    const text = await getTerminalText(sharedPage);
    const prompts = text.split("$ ").length - 1;
    expect(prompts).toBeGreaterThanOrEqual(2);
  });

  test("executes another whitelisted command", async () => {
    await executeCommand(sharedPage, "uname");
    const text = await getTerminalText(sharedPage);
    expect(text).toContain("Linux");
  });
});
