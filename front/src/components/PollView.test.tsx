import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import PollView from "./PollView";

/** Default props for rendering PollView in tests. */
const defaultProps = {
  options: ["A", "B", "C"],
  maxChoices: 1,
  votes: { A: 10, B: 5, C: 0 } as Record<string, number>,
  myChoices: [] as string[],
  onVote: vi.fn(),
  onUnvote: vi.fn(),
  onSwitch: vi.fn(),
};

describe("PollView", () => {
  it("renders all options", () => {
    render(<PollView {...defaultProps} />);
    expect(screen.getByTestId("poll-option-0")).toHaveTextContent("A");
    expect(screen.getByTestId("poll-option-1")).toHaveTextContent("B");
    expect(screen.getByTestId("poll-option-2")).toHaveTextContent("C");
  });

  it("shows vote counts for each option", () => {
    render(<PollView {...defaultProps} />);
    expect(screen.getByTestId("poll-count-0").textContent).toBe("10");
    expect(screen.getByTestId("poll-count-1").textContent).toBe("5");
    expect(screen.getByTestId("poll-count-2").textContent).toBe("0");
  });

  it("shows vote bar width proportional to votes", () => {
    render(<PollView {...defaultProps} />);
    const barA = screen.getByTestId("poll-bar-0");
    const barC = screen.getByTestId("poll-bar-2");
    expect(barA.style.width).toBe("66.66666666666666%");
    expect(barC.style.width).toBe("0%");
  });

  it("calls onVote when clicking unselected option", () => {
    const onVote = vi.fn();
    render(<PollView {...defaultProps} onVote={onVote} />);
    fireEvent.click(screen.getByTestId("poll-option-1"));
    expect(onVote).toHaveBeenCalledWith("B");
  });

  it("calls onUnvote when clicking already-selected option", () => {
    const onUnvote = vi.fn();
    render(<PollView {...defaultProps} myChoices={["A"]} onUnvote={onUnvote} />);
    fireEvent.click(screen.getByTestId("poll-option-0"));
    expect(onUnvote).toHaveBeenCalledWith("A");
  });

  it("calls onSwitch when clicking different option with maxChoices=1 and already voted", () => {
    const onSwitch = vi.fn();
    render(<PollView {...defaultProps} myChoices={["A"]} onSwitch={onSwitch} />);
    fireEvent.click(screen.getByTestId("poll-option-1"));
    expect(onSwitch).toHaveBeenCalledWith("A", "B");
  });

  it("allows multiple votes when maxChoices > 1", () => {
    const onVote = vi.fn();
    render(<PollView {...defaultProps} maxChoices={2} myChoices={["A"]} onVote={onVote} />);
    fireEvent.click(screen.getByTestId("poll-option-1"));
    expect(onVote).toHaveBeenCalledWith("B");
  });

  it("disables options when max choices reached with maxChoices > 1", () => {
    render(<PollView {...defaultProps} maxChoices={2} myChoices={["A", "B"]} />);
    expect(screen.getByTestId("poll-option-2")).toBeDisabled();
  });

  it("handles zero total votes without division error", () => {
    render(<PollView {...defaultProps} votes={{ A: 0, B: 0, C: 0 }} />);
    const bar = screen.getByTestId("poll-bar-0");
    expect(bar.style.width).toBe("0%");
  });

  it("renders with empty options array", () => {
    render(<PollView {...defaultProps} options={[]} />);
    expect(screen.getByTestId("poll-view")).toBeInTheDocument();
    expect(screen.queryByTestId("poll-option-0")).toBeNull();
  });

  it("highlights selected options visually", () => {
    render(<PollView {...defaultProps} myChoices={["A"]} />);
    const selected = screen.getByTestId("poll-option-0");
    const unselected = screen.getByTestId("poll-option-1");
    expect(selected.style.background).not.toBe(unselected.style.background);
  });
});
