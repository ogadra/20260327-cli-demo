import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import App from "./App";

vi.mock("./hooks/useSession", () => ({
  useSession: () => true,
}));

/** Default presenter state restored before each test. */
const defaultPresenter = {
  page: 0,
  viewerCount: 0,
  pollStates: {} as Record<string, unknown>,
};

const mockPresenter: {
  page: number;
  viewerCount: number;
  pollStates: Record<string, unknown>;
  sendPollVote: ReturnType<typeof vi.fn>;
  sendPollUnvote: ReturnType<typeof vi.fn>;
  sendPollSwitch: ReturnType<typeof vi.fn>;
} = {
  ...defaultPresenter,
  sendPollVote: vi.fn(),
  sendPollUnvote: vi.fn(),
  sendPollSwitch: vi.fn(),
};

vi.mock("./hooks/usePresenter", () => ({
  usePresenter: () => mockPresenter,
}));

vi.mock("./slides/index", () => ({
  slides: [() => <div>Slide Zero</div>, () => <div>Slide One</div>, () => <div>Slide Two</div>],
}));

const scrollIntoViewMock = vi.fn();

beforeEach(() => {
  Object.assign(mockPresenter, defaultPresenter);
  scrollIntoViewMock.mockClear();
  HTMLElement.prototype.scrollIntoView = scrollIntoViewMock;
});

describe("App", () => {
  it("renders all slides vertically", () => {
    render(<App />);
    expect(screen.getByText("Slide Zero")).toBeInTheDocument();
    expect(screen.getByText("Slide One")).toBeInTheDocument();
    expect(screen.getByText("Slide Two")).toBeInTheDocument();
  });

  it("displays viewer count", () => {
    mockPresenter.viewerCount = 42;
    render(<App />);
    expect(screen.getByText(/viewers/)).toHaveTextContent("42 viewers");
  });

  it("scrolls to the active page on initial render", () => {
    mockPresenter.page = 1;
    render(<App />);
    expect(scrollIntoViewMock).toHaveBeenCalled();
  });

  it("uses instant scroll for the first navigation", () => {
    mockPresenter.page = 2;
    render(<App />);
    expect(scrollIntoViewMock).toHaveBeenCalledWith(
      expect.objectContaining({ behavior: "instant" }),
    );
  });

  it("uses smooth scroll for subsequent navigations", () => {
    const { rerender } = render(<App />);
    scrollIntoViewMock.mockClear();
    mockPresenter.page = 1;
    rerender(<App />);
    expect(scrollIntoViewMock).toHaveBeenCalledWith(
      expect.objectContaining({ behavior: "smooth" }),
    );
  });
});
