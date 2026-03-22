import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import App from "./App";

vi.mock("./hooks/useSession", () => ({
  useSession: () => true,
}));

vi.mock("./hooks/useExecute", () => ({
  useExecute: () => ({ run: vi.fn(), running: false }),
}));

const mockWrite = vi.fn();

vi.mock("@xterm/xterm", () => {
  return {
    Terminal: class {
      write = mockWrite;
      writeln = vi.fn();
      open = vi.fn();
      dispose = vi.fn();
      loadAddon = vi.fn();
    },
  };
});

vi.mock("@xterm/addon-fit", () => {
  return {
    FitAddon: class {
      fit = vi.fn();
    },
  };
});

vi.mock("@xterm/xterm/css/xterm.css", () => ({}));

describe("App", () => {
  it("renders command input", () => {
    render(<App />);
    expect(screen.getByPlaceholderText("Enter command...")).toBeInTheDocument();
  });

  it("writes initial prompt when session is ready", () => {
    mockWrite.mockClear();
    render(<App />);
    expect(mockWrite).toHaveBeenCalledWith("$ ");
  });

  it("disables input when session is not ready", async () => {
    vi.resetModules();
    vi.doMock("./hooks/useSession", () => ({
      useSession: () => false,
    }));
    vi.doMock("./hooks/useExecute", () => ({
      useExecute: () => ({ run: vi.fn(), running: false }),
    }));
    const { default: AppNoSession } = await import("./App");
    render(<AppNoSession />);
    expect(screen.getByPlaceholderText("Enter command...")).toBeDisabled();
  });

  it("disables input when command is running", async () => {
    vi.resetModules();
    vi.doMock("./hooks/useSession", () => ({
      useSession: () => true,
    }));
    vi.doMock("./hooks/useExecute", () => ({
      useExecute: () => ({ run: vi.fn(), running: true }),
    }));
    const { default: AppRunning } = await import("./App");
    render(<AppRunning />);
    expect(screen.getByPlaceholderText("Enter command...")).toBeDisabled();
  });
});
