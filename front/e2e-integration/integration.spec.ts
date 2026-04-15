import { expect, test } from "@playwright/test";
import { Page } from "@playwright/test";

import { slideData } from "../src/slides/slideData";

/** Index of the first terminal slide in slideData, derived at build time. */
const TERMINAL_SLIDE_PAGE = slideData.findIndex((s) => s.type === "terminal");

/** CSS selector for the command input field in the terminal slide. */
const COMMAND_INPUT_SELECTOR = 'input[placeholder="date"]';

/** Get the locator for the terminal slide section. */
const terminalSection = (page: Page): ReturnType<Page["locator"]> =>
  page.locator(`div[role="region"]`).nth(TERMINAL_SLIDE_PAGE);

/** Wait for the command input to be enabled, indicating the session is ready. */
const waitForReady = async (page: Page): Promise<void> => {
  await expect(terminalSection(page).locator(COMMAND_INPUT_SELECTOR)).toBeEnabled({
    timeout: 30_000,
  });
};

/** Execute a command and wait for the input to be re-enabled after completion. */
const executeCommand = async (page: Page, command: string): Promise<void> => {
  const input = terminalSection(page).locator(COMMAND_INPUT_SELECTOR);
  await input.fill(command);
  await input.press("Enter");
  await expect(input).toBeEnabled({ timeout: 30_000 });
};

/** Get the text content of the terminal output from xterm.js. */
const getTerminalText = async (page: Page): Promise<string> =>
  (await terminalSection(page).locator(".xterm-rows").textContent()) ?? "";

/** Wait for the terminal text to change from the previously captured snapshot. */
const waitForTerminalChange = async (page: Page, previousText: string): Promise<string> => {
  const rows = terminalSection(page).locator(".xterm-rows");
  await expect(rows).not.toHaveText(previousText, { timeout: 10_000 });
  return (await rows.textContent()) ?? "";
};

/** Wait until the terminal contains at least the expected number of prompt markers. */
const waitForPromptCount = async (page: Page, count: number): Promise<string> => {
  const rows = terminalSection(page).locator(".xterm-rows");
  await expect(rows).toContainText("$ ", { timeout: 10_000 });
  let text = "";
  for (let i = 0; i < 50; i++) {
    text = (await rows.textContent()) ?? "";
    if (text.split("$ ").length - 1 >= count) return text;
    await page.waitForTimeout(200);
  }
  return text;
};

/** Mock the presenter WebSocket to navigate to the first terminal slide so CommandInput renders. */
const mockPresenterWs = async (page: Page): Promise<void> => {
  await page.routeWebSocket(/\/ws$/, (ws) => {
    ws.onMessage(() => {
      /* ignore outgoing messages */
    });
    ws.send(JSON.stringify({ type: "slide_sync", page: TERMINAL_SLIDE_PAGE }));
  });
};

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
