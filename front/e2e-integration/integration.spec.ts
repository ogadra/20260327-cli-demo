import { expect, test } from "@playwright/test";
import { Page } from "@playwright/test";

/** Wait for the command input to be enabled, indicating the session is ready. */
async function waitForReady(page: Page): Promise<void> {
  await expect(page.locator('input[placeholder="echo hello"]')).toBeEnabled({
    timeout: 30_000,
  });
}

/** Execute a command and wait for the input to be re-enabled after completion. */
async function executeCommand(page: Page, command: string): Promise<void> {
  const input = page.locator('input[placeholder="echo hello"]');
  await input.fill(command);
  await input.press("Enter");
  await expect(input).toBeEnabled({ timeout: 30_000 });
}

/** Get the text content of the terminal output from xterm.js. */
async function getTerminalText(page: Page): Promise<string> {
  return (await page.locator(".xterm-rows").textContent()) ?? "";
}

/** Wait for the terminal text to change from the previously captured snapshot. */
async function waitForTerminalChange(page: Page, previousText: string): Promise<string> {
  const rows = page.locator(".xterm-rows");
  await expect(rows).not.toHaveText(previousText, { timeout: 10_000 });
  return (await rows.textContent()) ?? "";
}

/** Wait until the terminal contains at least the expected number of prompt markers. */
async function waitForPromptCount(page: Page, count: number): Promise<string> {
  const rows = page.locator(".xterm-rows");
  await expect(rows).toContainText("$ ".repeat(1), { timeout: 10_000 });
  let text = "";
  for (let i = 0; i < 50; i++) {
    text = (await rows.textContent()) ?? "";
    if (text.split("$ ").length - 1 >= count) return text;
    await page.waitForTimeout(200);
  }
  return text;
}

/** Mock the presenter WebSocket to immediately send a hands_on message so CommandInput renders. */
async function mockPresenterWs(page: Page): Promise<void> {
  await page.routeWebSocket(/\/ws$/, (ws) => {
    ws.onMessage(() => {
      /* ignore outgoing messages */
    });
    ws.send(JSON.stringify({ type: "hands_on", instruction: "", placeholder: "" }));
  });
}

test.describe.serial("integration", () => {
  let sharedPage: Page;

  test.beforeAll(async ({ browser }) => {
    const context = await browser.newContext();
    sharedPage = await context.newPage();
    await mockPresenterWs(sharedPage);
    await sharedPage.goto("/");
    await waitForReady(sharedPage);
  });

  test.afterAll(async () => {
    await sharedPage.close();
  });

  test("executes command and shows output", async () => {
    const before1 = await getTerminalText(sharedPage);
    await executeCommand(sharedPage, "pwd");
    await waitForTerminalChange(sharedPage, before1);
    const text1 = await waitForPromptCount(sharedPage, 2);
    expect(text1, "Expected pwd command to display current directory").toMatch(/\//);

    const prompts = text1.split("$ ").length - 1;
    expect(prompts, "Expected at least 2 prompts").toBeGreaterThanOrEqual(2);

    const before2 = await getTerminalText(sharedPage);
    await executeCommand(sharedPage, "uname");
    const text2 = await waitForTerminalChange(sharedPage, before2);
    expect(text2, "Expected uname command to display OS information").toContain("Linux");
  });
});
