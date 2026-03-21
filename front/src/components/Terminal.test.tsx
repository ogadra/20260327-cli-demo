import { beforeEach, describe, expect, it, vi } from "vitest";
import { createRef } from "react";
import { render } from "@testing-library/react";
import Terminal, { type TerminalHandle } from "./Terminal";

const mockWrite = vi.fn();
const mockWriteln = vi.fn();
const mockOpen = vi.fn();
const mockDispose = vi.fn();
const mockLoadAddon = vi.fn();
const mockFit = vi.fn();

vi.mock("@xterm/xterm", () => {
  return {
    Terminal: class {
      write = mockWrite;
      writeln = mockWriteln;
      open = mockOpen;
      dispose = mockDispose;
      loadAddon = mockLoadAddon;
    },
  };
});

vi.mock("@xterm/addon-fit", () => {
  return {
    FitAddon: class {
      fit = mockFit;
    },
  };
});

vi.mock("@xterm/xterm/css/xterm.css", () => ({}));

beforeEach(() => {
  mockWrite.mockClear();
  mockWriteln.mockClear();
  mockOpen.mockClear();
  mockDispose.mockClear();
  mockLoadAddon.mockClear();
  mockFit.mockClear();
});

describe("Terminal", () => {
  it("initializes xterm on mount", () => {
    render(<Terminal />);
    expect(mockOpen).toHaveBeenCalledOnce();
  });

  it("exposes write and writeln via ref", () => {
    const ref = createRef<TerminalHandle>();
    render(<Terminal ref={ref} />);

    ref.current!.write("hello");
    expect(mockWrite).toHaveBeenCalledWith("hello");

    ref.current!.writeln("line");
    expect(mockWriteln).toHaveBeenCalledWith("line");
  });

  it("disposes xterm on unmount", () => {
    const { unmount } = render(<Terminal />);
    unmount();
    expect(mockDispose).toHaveBeenCalledOnce();
  });

  it("loads FitAddon and fits on mount", () => {
    render(<Terminal />);
    expect(mockLoadAddon).toHaveBeenCalledOnce();
    expect(mockFit).toHaveBeenCalledOnce();
  });

  it("refits on window resize", () => {
    render(<Terminal />);
    mockFit.mockClear();
    window.dispatchEvent(new Event("resize"));
    expect(mockFit).toHaveBeenCalledOnce();
  });
});
