import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import App from "./App";
import { MessageType, type PresenterMode } from "./api/presenter";

vi.mock("./hooks/useSession", () => ({
  useSession: () => true,
}));

vi.mock("./hooks/useExecute", () => ({
  useExecute: () => ({ run: vi.fn(), running: false }),
}));

const mockPresenter: {
  page: number;
  mode: PresenterMode;
  instruction: string;
  placeholder: string;
  viewerCount: number;
  sendSlideSync: ReturnType<typeof vi.fn>;
  sendHandsOn: ReturnType<typeof vi.fn>;
} = {
  page: 0,
  mode: MessageType.SlideSync,
  instruction: "",
  placeholder: "",
  viewerCount: 0,
  sendSlideSync: vi.fn(),
  sendHandsOn: vi.fn(),
};

vi.mock("./hooks/usePresenter", () => ({
  usePresenter: () => mockPresenter,
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

vi.mock("./slides/index", () => ({
  default: [() => <div>Test Slide</div>],
}));

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

  it("shows slide mode by default", () => {
    render(<App />);
    const slideMode = screen.getByTestId("slide-mode");
    expect(slideMode.style.display).toBe("flex");
    const handsOnMode = screen.getByTestId("hands-on-mode");
    expect(handsOnMode.style.display).toBe("none");
  });

  it("shows hands-on placeholder with default text", () => {
    render(<App />);
    expect(screen.getByTestId("hands-on-placeholder").textContent).toBe(
      "まもなくハンズオンが始まります",
    );
  });

  it("shows placeholder text when provided", () => {
    mockPresenter.placeholder = "$ echo hello";
    render(<App />);
    expect(screen.getByTestId("hands-on-placeholder").textContent).toBe("$ echo hello");
    mockPresenter.placeholder = "";
  });

  it("shows hands-on mode when mode is hands_on", () => {
    mockPresenter.mode = MessageType.HandsOn;
    render(<App />);
    const slideMode = screen.getByTestId("slide-mode");
    expect(slideMode.style.display).toBe("none");
    const handsOnMode = screen.getByTestId("hands-on-mode");
    expect(handsOnMode.style.display).toBe("flex");
    mockPresenter.mode = MessageType.SlideSync;
  });

  it("displays viewer count", () => {
    mockPresenter.viewerCount = 42;
    render(<App />);
    expect(screen.getByTestId("viewer-count").textContent).toBe("42 viewers");
    mockPresenter.viewerCount = 0;
  });

  it("places input as first child in hands-on mode", () => {
    mockPresenter.mode = MessageType.HandsOn;
    render(<App />);
    const handsOnMode = screen.getByTestId("hands-on-mode");
    const firstChild = handsOnMode.children[0] as HTMLElement;
    expect(firstChild.tagName).toBe("INPUT");
    expect(firstChild.getAttribute("placeholder")).toBe("Enter command...");
    mockPresenter.mode = MessageType.SlideSync;
  });

  it("disables input when session is not ready", async () => {
    vi.resetModules();
    vi.doMock("./hooks/useSession", () => ({
      useSession: () => false,
    }));
    vi.doMock("./hooks/useExecute", () => ({
      useExecute: () => ({ run: vi.fn(), running: false }),
    }));
    vi.doMock("./hooks/usePresenter", () => ({
      usePresenter: () => mockPresenter,
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
    vi.doMock("./hooks/usePresenter", () => ({
      usePresenter: () => mockPresenter,
    }));
    const { default: AppRunning } = await import("./App");
    render(<AppRunning />);
    expect(screen.getByPlaceholderText("Enter command...")).toBeDisabled();
  });
});
