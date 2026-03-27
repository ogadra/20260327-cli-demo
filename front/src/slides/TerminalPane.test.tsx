import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { TerminalPane } from "./TerminalPane";

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

const mockRun = vi.fn();
vi.mock("../hooks/useExecute", () => ({
  useExecute: () => ({ run: mockRun, running: false }),
}));

describe("TerminalPane", () => {
  it("renders command input with placeholder", () => {
    render(<TerminalPane ready={true} placeholder="echo hello" />);
    expect(screen.getByPlaceholderText("echo hello")).toBeDefined();
  });

  it("renders run button", () => {
    render(<TerminalPane ready={true} placeholder="ls" />);
    expect(screen.getByRole("button", { name: "Run" })).toBeDefined();
  });

  it("disables input when not ready", () => {
    render(<TerminalPane ready={false} placeholder="ls" />);
    expect(screen.getByPlaceholderText("ls").hasAttribute("disabled")).toBe(true);
  });
});
