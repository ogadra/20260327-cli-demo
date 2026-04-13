import { expect, test } from "@playwright/test";
import { Page } from "@playwright/test";

/** Index of the first terminal slide in slideData. */
const TERMINAL_SLIDE_PAGE = 26;

/** CSS selector for the command input field on terminal slides. */
const COMMAND_INPUT_SELECTOR = 'input[placeholder="date"]';

/** Wait for the command input to be enabled, indicating the session is ready. */
async function waitForReady(page: Page): Promise<void> {
  await expect(page.locator(COMMAND_INPUT_SELECTOR)).toBeEnabled({
    timeout: 30_000,
  });
}

/** Execute a command and wait for the input to be re-enabled after completion. */
async function executeCommand(page: Page, command: string): Promise<void> {
  const input = page.locator(COMMAND_INPUT_SELECTOR);
  await input.fill(command);
  await input.press("Enter");
  await expect(input).toBeEnabled({ timeout: 30_000 });
}

/** Locate the .xterm-rows element scoped to the active terminal slide container. */
function terminalRows(page: Page): ReturnType<Page["locator"]> {
  return page.locator(COMMAND_INPUT_SELECTOR).locator("../..").locator(".xterm-rows").first();
}

/** Get the text content of the terminal output from xterm.js. */
async function getTerminalText(page: Page): Promise<string> {
  return (await terminalRows(page).textContent()) ?? "";
}

/** Wait for the terminal text to change from the previously captured snapshot. */
async function waitForTerminalChange(page: Page, previousText: string): Promise<string> {
  const rows = terminalRows(page);
  await expect(rows).not.toHaveText(previousText, { timeout: 10_000 });
  return (await rows.textContent()) ?? "";
}

/** Wait until the terminal contains at least the expected number of prompt markers. */
async function waitForPromptCount(page: Page, count: number): Promise<string> {
  const rows = terminalRows(page);
  await expect(rows).toContainText("$ ", { timeout: 10_000 });
  let text = "";
  for (let i = 0; i < 50; i++) {
    text = (await rows.textContent()) ?? "";
    if (text.split("$ ").length - 1 >= count) return text;
    await page.waitForTimeout(200);
  }
  return text;
}

/** Mock the presenter WebSocket to navigate to the first terminal slide so CommandInput renders. */
async function mockPresenterWs(page: Page): Promise<void> {
  await page.routeWebSocket(/\/ws$/, (ws) => {
    ws.onMessage(() => {
      /* ignore outgoing messages */
    });
    ws.send(JSON.stringify({ type: "slide_sync", page: TERMINAL_SLIDE_PAGE }));
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
