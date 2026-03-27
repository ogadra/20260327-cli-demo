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

  it("writes prompt to terminal when status is ready", () => {
    renderSlide("ready");
    expect(mockWrite).toHaveBeenCalledWith("$ ");
    expect(mockWriteln).not.toHaveBeenCalled();
  });

  it("renders instruction text", () => {
    renderSlide("loading");
    expect(screen.getByText("Try a command")).toBeDefined();
  });

  it("disables command input when not ready", () => {
    renderSlide("loading");
    const input = screen.getByPlaceholderText("echo hello");
    expect(input).toBeDisabled();
  });

  it("enables command input when ready", () => {
    renderSlide("ready");
    const input = screen.getByPlaceholderText("echo hello");
    expect(input).not.toBeDisabled();
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
    expect(mockWrite).toHaveBeenCalledWith("$ ");
  });
});
