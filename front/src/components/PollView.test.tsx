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
    expect(screen.getByRole("button", { name: /^A/ })).toHaveTextContent("A");
    expect(screen.getByRole("button", { name: /^B/ })).toHaveTextContent("B");
    expect(screen.getByRole("button", { name: /^C/ })).toHaveTextContent("C");
  });

  it("shows vote counts for each option", () => {
    render(<PollView {...defaultProps} />);
    expect(screen.getByRole("button", { name: /^A/ }).textContent).toContain("10");
    expect(screen.getByRole("button", { name: /^B/ }).textContent).toContain("5");
    expect(screen.getByRole("button", { name: /^C/ }).textContent).toContain("0");
  });

  it("shows vote bar width proportional to votes", () => {
    render(<PollView {...defaultProps} />);
    const btnA = screen.getByRole("button", { name: /^A/ });
    const btnC = screen.getByRole("button", { name: /^C/ });
    const barA = btnA.querySelector("[aria-hidden]") as HTMLElement;
    const barC = btnC.querySelector("[aria-hidden]") as HTMLElement;
    expect(parseFloat(barA.style.width)).toBeCloseTo(66.6667, 3);
    expect(barC.style.width).toBe("0%");
  });

  it("calls onVote when clicking unselected option", () => {
    const onVote = vi.fn();
    render(<PollView {...defaultProps} onVote={onVote} />);
    fireEvent.click(screen.getByRole("button", { name: /^B/ }));
    expect(onVote).toHaveBeenCalledWith("B");
  });

  it("does nothing when clicking already-selected option with maxChoices=1", () => {
    const onUnvote = vi.fn();
    render(<PollView {...defaultProps} myChoices={["A"]} onUnvote={onUnvote} />);
    fireEvent.click(screen.getByRole("button", { name: /^A/ }));
    expect(onUnvote).not.toHaveBeenCalled();
  });

  it("calls onUnvote when clicking already-selected option with maxChoices > 1", () => {
    const onUnvote = vi.fn();
    render(<PollView {...defaultProps} maxChoices={2} myChoices={["A"]} onUnvote={onUnvote} />);
    fireEvent.click(screen.getByRole("button", { name: /^A/ }));
    expect(onUnvote).toHaveBeenCalledWith("A");
  });

  it("calls onSwitch when clicking different option with maxChoices=1 and already voted", () => {
    const onSwitch = vi.fn();
    render(<PollView {...defaultProps} myChoices={["A"]} onSwitch={onSwitch} />);
    fireEvent.click(screen.getByRole("button", { name: /^B/ }));
    expect(onSwitch).toHaveBeenCalledWith("A", "B");
  });

  it("allows multiple votes when maxChoices > 1", () => {
    const onVote = vi.fn();
    render(<PollView {...defaultProps} maxChoices={2} myChoices={["A"]} onVote={onVote} />);
    fireEvent.click(screen.getByRole("button", { name: /^B/ }));
    expect(onVote).toHaveBeenCalledWith("B");
  });

  it("disables options when max choices reached with maxChoices > 1", () => {
    render(<PollView {...defaultProps} maxChoices={2} myChoices={["A", "B"]} />);
    expect(screen.getByRole("button", { name: /^C/ })).toBeDisabled();
  });

  it("handles zero total votes without division error", () => {
    render(<PollView {...defaultProps} votes={{ A: 0, B: 0, C: 0 }} />);
    const btnA = screen.getByRole("button", { name: /^A/ });
    const bar = btnA.querySelector("[aria-hidden]") as HTMLElement;
    expect(bar.style.width).toBe("0%");
  });

  it("renders with empty options array", () => {
    render(<PollView {...defaultProps} options={[]} />);
    expect(screen.getByRole("group")).toBeInTheDocument();
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("ignores votes for keys not in options when computing bar width", () => {
    render(<PollView {...defaultProps} options={["A", "B"]} votes={{ A: 10, B: 10, Z: 80 }} />);
    const btnA = screen.getByRole("button", { name: /^A/ });
    const barA = btnA.querySelector("[aria-hidden]") as HTMLElement;
    expect(parseFloat(barA.style.width)).toBeCloseTo(50, 0);
  });

  it("highlights selected options visually", () => {
    render(<PollView {...defaultProps} myChoices={["A"]} />);
    const selected = screen.getByRole("button", { name: /^A/ });
    const unselected = screen.getByRole("button", { name: /^B/ });
    expect(selected.style.background).not.toBe(unselected.style.background);
  });
});
