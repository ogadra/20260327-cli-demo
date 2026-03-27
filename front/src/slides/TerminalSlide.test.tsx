import { forwardRef, useImperativeHandle } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { TerminalSlide } from "./TerminalSlide";
import type { SessionStatus } from "../hooks/useSession";

const mockWrite = vi.fn();
const mockWriteln = vi.fn();

vi.mock("../components/Terminal", () => ({
  default: forwardRef((_: unknown, ref: React.Ref<unknown>) => {
    useImperativeHandle(ref, () => ({ write: mockWrite, writeln: mockWriteln }));
    return <div data-testid="terminal" />;
  }),
}));

vi.mock("../hooks/useExecute", () => ({
  useExecute: () => ({ run: vi.fn(), running: false }),
}));

beforeEach(() => {
  mockWrite.mockClear();
  mockWriteln.mockClear();
});

/** Default props for TerminalSlide tests. */
const defaultProps = {
  instruction: "Try a command",
  commands: ["echo hello"],
};

/** Render helper that merges sessionStatus with default props. */
const renderSlide = (sessionStatus: SessionStatus): ReturnType<typeof render> =>
  render(<TerminalSlide sessionStatus={sessionStatus} {...defaultProps} />);

describe("TerminalSlide", () => {
  it("writes loading message to terminal when status is loading", () => {
    renderSlide("loading");
    expect(mockWriteln).toHaveBeenCalledWith("Connecting to runner...");
    expect(mockWrite).not.toHaveBeenCalled();
  });

  it("writes retrying message with yellow ANSI color when status is retrying", () => {
    renderSlide("retrying");
    expect(mockWriteln).toHaveBeenCalledWith("\x1b[33mConnection failed. Retrying...\x1b[0m");
    expect(mockWrite).not.toHaveBeenCalled();
  });

  it("does not write prompt from TerminalPane when status is ready since useExecute handles it", () => {
    renderSlide("ready");
    expect(mockWriteln).not.toHaveBeenCalled();
  });

  it("renders instruction text", () => {
    renderSlide("loading");
    expect(screen.getByText("Try a command")).toBeDefined();
  });

  it("does not render instruction when empty string", () => {
    render(<TerminalSlide sessionStatus="loading" instruction="" commands={["echo hi"]} />);
    expect(screen.queryByText("Try a command")).toBeNull();
  });

  it("renders one terminal pane for single command", () => {
    renderSlide("ready");
    const buttons = screen.getAllByRole("button", { name: "Run" });
    expect(buttons).toHaveLength(1);
  });

  it("renders multiple terminal panes for multiple commands", () => {
    render(
      <TerminalSlide sessionStatus="ready" instruction="" commands={["date", "whoami", "ls"]} />,
    );
    const buttons = screen.getAllByRole("button", { name: "Run" });
    expect(buttons).toHaveLength(3);
  });

  it("shows correct placeholders for each command", () => {
    render(<TerminalSlide sessionStatus="ready" instruction="" commands={["date", "whoami"]} />);
    expect(screen.getByPlaceholderText("date")).toBeDefined();
    expect(screen.getByPlaceholderText("whoami")).toBeDefined();
  });

  it("updates terminal message when status transitions", () => {
    const { rerender } = render(<TerminalSlide sessionStatus="loading" {...defaultProps} />);
    expect(mockWriteln).toHaveBeenCalledWith("Connecting to runner...");

    mockWrite.mockClear();
    mockWriteln.mockClear();

    rerender(<TerminalSlide sessionStatus="retrying" {...defaultProps} />);
    expect(mockWriteln).toHaveBeenCalledWith("\x1b[33mConnection failed. Retrying...\x1b[0m");

    mockWrite.mockClear();
    mockWriteln.mockClear();

    rerender(<TerminalSlide sessionStatus="ready" {...defaultProps} />);
    expect(mockWriteln).not.toHaveBeenCalled();
  });
});
