import { beforeEach, describe, expect, it, type Mock, vi } from "vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import type { PresenterPanelProps } from "./PresenterPanel";

vi.mock("./sequence", async () => {
  const { Action } = await vi.importActual<typeof import("../api/presenter")>("../api/presenter");
  return {
    defaultSequence: [
      { type: Action.SlideSync, page: 0 },
      { type: Action.HandsOn, instruction: "Run echo", placeholder: "echo hello" },
      { type: Action.PollOpen, pollId: "q1", options: ["Yes", "No"], maxChoices: 1 },
      { type: Action.SlideSync, page: 1 },
    ],
  };
});

/** Creates a fresh set of props with mock send functions for each test. */
const createProps = (): PresenterPanelProps & {
  sendSlideSync: Mock;
  sendHandsOn: Mock;
  sendPollGet: Mock;
} => ({
  sendSlideSync: vi.fn(),
  sendHandsOn: vi.fn(),
  sendPollGet: vi.fn(),
  viewerCount: 0,
  pollStates: {},
});

describe("PresenterPanel", () => {
  let props: ReturnType<typeof createProps>;

  beforeEach(() => {
    props = createProps();
  });

  /** Lazily imports PresenterPanel to pick up the mocked sequence. */
  const renderPanel = async (): Promise<void> => {
    const { PresenterPanel } = await import("./PresenterPanel");
    render(<PresenterPanel {...props} />);
  };

  it("renders step counter", async () => {
    await renderPanel();
    const nav = screen.getByRole("navigation", { name: "status" });
    expect(within(nav).getByText(/Step 1 \/ 4/)).toBeTruthy();
  });

  it("shows viewer count", async () => {
    props.viewerCount = 42;
    await renderPanel();
    expect(screen.getByText("42 viewers")).toBeTruthy();
  });

  it("disables prev button at step 0", async () => {
    await renderPanel();
    expect(screen.getByRole("button", { name: "Prev" }).hasAttribute("disabled")).toBe(true);
  });

  it("enables next button at step 0", async () => {
    await renderPanel();
    expect(screen.getByRole("button", { name: "Next" }).hasAttribute("disabled")).toBe(false);
  });

  it("advances step and calls sendHandsOn on next", async () => {
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(props.sendHandsOn).toHaveBeenCalledWith("Run echo", "echo hello");
    expect(screen.getByText(/Step 2 \/ 4/)).toBeTruthy();
  });

  it("calls sendPollGet when navigating to poll step", async () => {
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(props.sendPollGet).toHaveBeenCalledWith("q1", ["Yes", "No"], 1);
    expect(screen.getByText(/Step 3 \/ 4/)).toBeTruthy();
  });

  it("goes back with prev button", async () => {
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Prev" }));
    expect(screen.getByText(/Step 1 \/ 4/)).toBeTruthy();
  });

  it("disables next button at last step", async () => {
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByRole("button", { name: "Next" }).hasAttribute("disabled")).toBe(true);
    expect(screen.getByText(/Step 4 \/ 4/)).toBeTruthy();
  });

  it("advances step on ArrowRight key", async () => {
    await renderPanel();
    fireEvent.keyDown(window, { key: "ArrowRight" });
    expect(props.sendHandsOn).toHaveBeenCalledWith("Run echo", "echo hello");
    expect(screen.getByText(/Step 2 \/ 4/)).toBeTruthy();
  });

  it("goes back on ArrowLeft key", async () => {
    await renderPanel();
    fireEvent.keyDown(window, { key: "ArrowRight" });
    fireEvent.keyDown(window, { key: "ArrowLeft" });
    expect(screen.getByText(/Step 1 \/ 4/)).toBeTruthy();
  });

  it("does not go below step 0 on ArrowLeft", async () => {
    await renderPanel();
    fireEvent.keyDown(window, { key: "ArrowLeft" });
    expect(screen.getByText(/Step 1 \/ 4/)).toBeTruthy();
  });

  it("does not go beyond last step on ArrowRight", async () => {
    await renderPanel();
    fireEvent.keyDown(window, { key: "ArrowRight" });
    fireEvent.keyDown(window, { key: "ArrowRight" });
    fireEvent.keyDown(window, { key: "ArrowRight" });
    fireEvent.keyDown(window, { key: "ArrowRight" });
    expect(screen.getByText(/Step 4 \/ 4/)).toBeTruthy();
  });

  it("executes step 0 on mount", async () => {
    await renderPanel();
    expect(props.sendSlideSync).toHaveBeenCalledWith(0);
  });

  it("displays step description for slide_sync step", async () => {
    await renderPanel();
    expect(screen.getByText("Slide 0")).toBeTruthy();
  });

  it("displays step description for hands_on step", async () => {
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText("Hands-on: Run echo")).toBeTruthy();
  });

  it("displays step description for poll_open step", async () => {
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText("Poll: q1")).toBeTruthy();
  });

  it("shows poll results when available for poll_open step", async () => {
    props.pollStates = {
      q1: {
        options: ["Yes", "No"],
        maxChoices: 1,
        votes: { Yes: 10, No: 5 },
        myChoices: [],
      },
    };
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    const results = screen.getByRole("region", { name: "poll results" });
    expect(results).toBeTruthy();
    expect(results.textContent).toContain("Yes");
    expect(results.textContent).toContain("10");
    expect(results.textContent).toContain("No");
    expect(results.textContent).toContain("5");
  });

  it("does not show poll results on non-poll steps", async () => {
    await renderPanel();
    expect(screen.queryByRole("region", { name: "poll results" })).toBeNull();
  });

  it("does not show poll results when pollStates has no data for the poll", async () => {
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.queryByRole("region", { name: "poll results" })).toBeNull();
  });
});
