import { forwardRef, useImperativeHandle } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { TerminalPane } from "./TerminalPane";
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

/** Render helper that creates a TerminalPane with given session status. */
const renderPane = (sessionStatus: SessionStatus): ReturnType<typeof render> =>
  render(<TerminalPane sessionStatus={sessionStatus} placeholder="echo hello" />);

describe("TerminalPane", () => {
  it("renders command input with placeholder", () => {
    renderPane("ready");
    expect(screen.getByPlaceholderText("echo hello")).toBeDefined();
  });

  it("renders run button", () => {
    renderPane("ready");
    expect(screen.getByRole("button", { name: "Run" })).toBeDefined();
  });

  it("disables input when not ready", () => {
    renderPane("loading");
    expect(screen.getByPlaceholderText("echo hello")).toBeDisabled();
  });

  it("writes loading message to terminal when status is loading", () => {
    renderPane("loading");
    expect(mockWriteln).toHaveBeenCalledWith("Connecting to runner...");
  });

  it("writes retrying message to terminal when status is retrying", () => {
    renderPane("retrying");
    expect(mockWriteln).toHaveBeenCalledWith("\x1b[33mConnection failed. Retrying...\x1b[0m");
  });

  it("does not write prompt to terminal when status is ready since useExecute handles it", () => {
    renderPane("ready");
    expect(mockWrite).not.toHaveBeenCalledWith("$ ");
  });
});
