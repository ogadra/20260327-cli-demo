import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { TerminalSlide } from "./TerminalSlide";

vi.mock("@xterm/xterm", () => ({
  Terminal: class {
    write = vi.fn();
    writeln = vi.fn();
    open = vi.fn();
    dispose = vi.fn();
    loadAddon = vi.fn();
  },
}));

vi.mock("@xterm/addon-fit", () => ({
  FitAddon: class {
    fit = vi.fn();
  },
}));

vi.mock("@xterm/xterm/css/xterm.css", () => ({}));

vi.mock("../hooks/useSession", () => ({
  useSession: () => true,
}));

vi.mock("../hooks/useExecute", () => ({
  useExecute: () => ({ run: vi.fn(), running: false }),
}));

describe("TerminalSlide", () => {
  it("renders instruction text when provided", () => {
    render(<TerminalSlide instruction="Try this" commands={["echo hi"]} />);
    expect(screen.getByText("Try this")).toBeDefined();
  });

  it("does not render instruction when empty string", () => {
    render(<TerminalSlide instruction="" commands={["echo hi"]} />);
    expect(screen.queryByText("Try this")).toBeNull();
  });

  it("renders one terminal pane for single command", () => {
    render(<TerminalSlide instruction="" commands={["echo hello"]} />);
    const buttons = screen.getAllByRole("button", { name: "Run" });
    expect(buttons).toHaveLength(1);
  });

  it("renders multiple terminal panes for multiple commands", () => {
    render(<TerminalSlide instruction="" commands={["date", "whoami", "ls"]} />);
    const buttons = screen.getAllByRole("button", { name: "Run" });
    expect(buttons).toHaveLength(3);
  });

  it("shows correct placeholders for each command", () => {
    render(<TerminalSlide instruction="" commands={["date", "whoami"]} />);
    expect(screen.getByPlaceholderText("date")).toBeDefined();
    expect(screen.getByPlaceholderText("whoami")).toBeDefined();
  });
});
