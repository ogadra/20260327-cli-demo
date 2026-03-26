import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import App from "./App";
import { ServerMessageType, type PresenterMode } from "./api/presenter";

vi.mock("./hooks/useSession", () => ({
  useSession: () => true,
}));

vi.mock("./hooks/useExecute", () => ({
  useExecute: () => ({ run: vi.fn(), running: false }),
}));

/** Default presenter state restored before each test. */
const defaultPresenter = {
  page: 0,
  mode: ServerMessageType.SlideSync as PresenterMode,
  instruction: "",
  placeholder: "",
  viewerCount: 0,
  pollStates: {} as Record<string, unknown>,
};

const mockPresenter: {
  page: number;
  mode: PresenterMode;
  instruction: string;
  placeholder: string;
  viewerCount: number;
  pollStates: Record<string, unknown>;
  sendSlideSync: ReturnType<typeof vi.fn>;
  sendHandsOn: ReturnType<typeof vi.fn>;
  sendPollVote: ReturnType<typeof vi.fn>;
  sendPollUnvote: ReturnType<typeof vi.fn>;
  sendPollSwitch: ReturnType<typeof vi.fn>;
} = {
  ...defaultPresenter,
  sendSlideSync: vi.fn(),
  sendHandsOn: vi.fn(),
  sendPollVote: vi.fn(),
  sendPollUnvote: vi.fn(),
  sendPollSwitch: vi.fn(),
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
  slides: [() => <div>Test Slide</div>],
}));

beforeEach(() => {
  Object.assign(mockPresenter, defaultPresenter);
  mockWrite.mockClear();
});

describe("App", () => {
  it("renders command input in hands-on mode", () => {
    mockPresenter.mode = ServerMessageType.HandsOn;
    render(<App />);
    expect(screen.getByPlaceholderText("Enter command...")).toBeInTheDocument();
  });

  it("does not render command input in slide mode", () => {
    render(<App />);
    expect(screen.queryByPlaceholderText("Enter command...")).toBeNull();
  });

  it("writes initial prompt when session is ready", () => {
    render(<App />);
    expect(mockWrite).toHaveBeenCalledWith("$ ");
  });

  it("shows slide mode by default", () => {
    render(<App />);
    const slideMode = screen.getByRole("region", { name: "Slide mode" });
    expect(slideMode.style.display).toBe("flex");
    const handsOnMode = document.querySelector('[aria-label="Hands-on mode"]') as HTMLElement;
    expect(handsOnMode.style.display).toBe("none");
  });

  it("hides instruction block when instruction is empty", () => {
    render(<App />);
    expect(screen.queryByText(/echo/)).toBeNull();
  });

  it("shows instruction block when instruction is provided", () => {
    mockPresenter.instruction = "echo hello を実行してみよう";
    render(<App />);
    expect(screen.getByText("echo hello を実行してみよう")).toBeInTheDocument();
  });

  it("passes placeholder to command input", () => {
    mockPresenter.placeholder = "$ echo hello";
    mockPresenter.mode = ServerMessageType.HandsOn;
    render(<App />);
    expect(screen.getByPlaceholderText("$ echo hello")).toBeInTheDocument();
  });

  it("shows hands-on mode when mode is hands_on", () => {
    mockPresenter.mode = ServerMessageType.HandsOn;
    render(<App />);
    const slideMode = document.querySelector('[aria-label="Slide mode"]') as HTMLElement;
    expect(slideMode.style.display).toBe("none");
    const handsOnMode = screen.getByRole("region", { name: "Hands-on mode" });
    expect(handsOnMode.style.display).toBe("flex");
  });

  it("displays viewer count", () => {
    mockPresenter.viewerCount = 42;
    render(<App />);
    expect(screen.getByText(/viewers/)).toHaveTextContent("42 viewers");
  });

  it("places input as first child in hands-on mode", () => {
    mockPresenter.mode = ServerMessageType.HandsOn;
    render(<App />);
    const handsOnMode = screen.getByRole("region", { name: "Hands-on mode" });
    const firstChild = handsOnMode.children[0] as HTMLElement;
    expect(firstChild.tagName).toBe("INPUT");
    expect(firstChild.getAttribute("placeholder")).toBe("Enter command...");
  });

  it("disables input when session is not ready", async () => {
    vi.resetModules();
    vi.doMock("./hooks/useSession", () => ({
      useSession: () => false,
    }));
    vi.doMock("./hooks/useExecute", () => ({
      useExecute: () => ({ run: vi.fn(), running: false }),
    }));
    mockPresenter.mode = ServerMessageType.HandsOn;
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
    mockPresenter.mode = ServerMessageType.HandsOn;
    vi.doMock("./hooks/usePresenter", () => ({
      usePresenter: () => mockPresenter,
    }));
    const { default: AppRunning } = await import("./App");
    render(<AppRunning />);
    expect(screen.getByPlaceholderText("Enter command...")).toBeDisabled();
  });
});
