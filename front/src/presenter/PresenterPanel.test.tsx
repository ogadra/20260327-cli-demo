import { beforeEach, describe, expect, it, type Mock, vi } from "vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import type { PresenterPanelProps } from "./PresenterPanel";

vi.mock("./sequence", async () => {
  const { Action } = await vi.importActual<typeof import("../api/presenter")>("../api/presenter");
  return {
    defaultSequence: [
      { type: Action.SlideSync, page: 0 },
      { type: Action.SlideSync, page: 1 },
      { type: Action.SlideSync, page: 2 },
    ],
    defaultPolls: [],
  };
});

/** Creates a fresh set of props with mock send functions for each test. */
const createProps = (): PresenterPanelProps & {
  sendSlideSync: Mock;
} => ({
  sendSlideSync: vi.fn(),
  viewerCount: 0,
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

  /** Verify that the step counter renders with the correct total. */
  it("renders step counter", async () => {
    await renderPanel();
    const status = screen.getByRole("status");
    expect(within(status).getByText(/Step 1 \/ 3/)).toBeTruthy();
  });

  /** Verify that the viewer count is displayed. */
  it("shows viewer count", async () => {
    props.viewerCount = 42;
    await renderPanel();
    expect(screen.getByText("42 viewers")).toBeTruthy();
  });

  /** Verify that the prev button is disabled at step 0. */
  it("disables prev button at step 0", async () => {
    await renderPanel();
    expect(screen.getByRole("button", { name: "Prev" }).hasAttribute("disabled")).toBe(true);
  });

  /** Verify that the next button is enabled at step 0. */
  it("enables next button at step 0", async () => {
    await renderPanel();
    expect(screen.getByRole("button", { name: "Next" }).hasAttribute("disabled")).toBe(false);
  });

  /** Verify that advancing calls sendSlideSync for the next step. */
  it("advances step and calls sendSlideSync on next", async () => {
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(props.sendSlideSync).toHaveBeenCalledWith(1);
    expect(screen.getByText(/Step 2 \/ 3/)).toBeTruthy();
  });

  /** Verify that the prev button navigates back. */
  it("goes back with prev button", async () => {
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Prev" }));
    expect(screen.getByText(/Step 1 \/ 3/)).toBeTruthy();
  });

  /** Verify that the next button is disabled at the last step. */
  it("disables next button at last step", async () => {
    await renderPanel();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByRole("button", { name: "Next" }).hasAttribute("disabled")).toBe(true);
    expect(screen.getByText(/Step 3 \/ 3/)).toBeTruthy();
  });

  /** Verify that ArrowRight advances the step. */
  it("advances step on ArrowRight key", async () => {
    await renderPanel();
    fireEvent.keyDown(window, { key: "ArrowRight" });
    expect(props.sendSlideSync).toHaveBeenCalledWith(1);
    expect(screen.getByText(/Step 2 \/ 3/)).toBeTruthy();
  });

  /** Verify that ArrowLeft navigates back. */
  it("goes back on ArrowLeft key", async () => {
    await renderPanel();
    fireEvent.keyDown(window, { key: "ArrowRight" });
    fireEvent.keyDown(window, { key: "ArrowLeft" });
    expect(screen.getByText(/Step 1 \/ 3/)).toBeTruthy();
  });

  /** Verify that ArrowLeft does not go below step 0. */
  it("does not go below step 0 on ArrowLeft", async () => {
    await renderPanel();
    fireEvent.keyDown(window, { key: "ArrowLeft" });
    expect(screen.getByText(/Step 1 \/ 3/)).toBeTruthy();
  });

  /** Verify that ArrowRight does not go beyond the last step. */
  it("does not go beyond last step on ArrowRight", async () => {
    await renderPanel();
    fireEvent.keyDown(window, { key: "ArrowRight" });
    fireEvent.keyDown(window, { key: "ArrowRight" });
    fireEvent.keyDown(window, { key: "ArrowRight" });
    expect(screen.getByText(/Step 3 \/ 3/)).toBeTruthy();
  });

  /** Verify that step 0 is executed on mount. */
  it("executes step 0 on mount", async () => {
    await renderPanel();
    expect(props.sendSlideSync).toHaveBeenCalledWith(0);
    expect(props.sendSlideSync).toHaveBeenCalledTimes(1);
  });

  /** Verify that slide step shows the correct description. */
  it("displays step description", async () => {
    await renderPanel();
    expect(screen.getByText("Slide 0")).toBeTruthy();
  });
});
