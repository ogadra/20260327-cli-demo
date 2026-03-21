import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import App from "./App";

vi.mock("./hooks/useSession", () => ({
  useSession: () => "test-session",
}));

vi.mock("./hooks/useExecute", () => ({
  useExecute: () => ({ run: vi.fn(), running: false }),
}));

vi.mock("@xterm/xterm", () => {
  return {
    Terminal: class {
      write = vi.fn();
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
    expect(screen.getByPlaceholderText("Enter command...")).toBeTruthy();
  });
});
