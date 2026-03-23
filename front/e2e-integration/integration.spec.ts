import { expect, test } from "@playwright/test";
import { Page } from "@playwright/test";

/** Wait for the command input to be enabled, indicating the session is ready. */
async function waitForReady(page: Page): Promise<void> {
  await expect(page.locator('input[placeholder="Enter command..."]')).toBeEnabled({
    timeout: 30_000,
  });
}

/** Execute a command and wait for the input to be re-enabled after completion. */
async function executeCommand(page: Page, command: string): Promise<void> {
  const input = page.locator('input[placeholder="Enter command..."]');
  await input.fill(command);
  await input.press("Enter");
  await expect(input).toBeEnabled({ timeout: 30_000 });
}

/** Get the text content of the terminal output from xterm.js. */
async function getTerminalText(page: Page): Promise<string> {
  return (await page.locator(".xterm-rows").textContent()) ?? "";
}

test.describe.serial("integration", () => {
  let sharedPage: Page;

  test.beforeAll(async ({ browser }) => {
    const context = await browser.newContext();
    sharedPage = await context.newPage();
    await sharedPage.goto("/");
    await waitForReady(sharedPage);
  });

  test.afterAll(async () => {
    await sharedPage.close();
  });

  test("executes command and shows output", async () => {
    await executeCommand(sharedPage, "pwd");
    const text1 = await getTerminalText(sharedPage);
    expect(text1, "Expected pwd command to display current directory").toMatch(/\//);

    const text2 = await getTerminalText(sharedPage);
    const prompts = text2.split("$ ").length - 1;
    expect(prompts, "Expected at least 2 prompts").toBeGreaterThanOrEqual(2);

    await executeCommand(sharedPage, "uname");
    const text3 = await getTerminalText(sharedPage);
    expect(text3, "Expected uname command to display OS information").toContain("Linux");
  });
});
