import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import CommandInput from "./CommandInput";

let rafCallback: FrameRequestCallback | null = null;
beforeEach(() => {
  rafCallback = null;
  vi.spyOn(window, "requestAnimationFrame").mockImplementation((cb) => {
    rafCallback = cb;
    return 0;
  });
});

describe("CommandInput", () => {
  it("calls onSubmit with value on Enter", () => {
    const onSubmit = vi.fn();
    render(<CommandInput onSubmit={onSubmit} disabled={false} />);

    const input = screen.getByPlaceholderText("Enter command...");
    fireEvent.change(input, { target: { value: "ls" } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(onSubmit).toHaveBeenCalledWith("ls");
    expect(input).toHaveValue("");
  });

  it("does not submit empty input", () => {
    const onSubmit = vi.fn();
    render(<CommandInput onSubmit={onSubmit} disabled={false} />);

    const input = screen.getByPlaceholderText("Enter command...");
    fireEvent.keyDown(input, { key: "Enter" });

    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("navigates history with arrow keys", () => {
    const onSubmit = vi.fn();
    render(<CommandInput onSubmit={onSubmit} disabled={false} />);

    const input = screen.getByPlaceholderText("Enter command...");

    fireEvent.change(input, { target: { value: "cmd1" } });
    fireEvent.keyDown(input, { key: "Enter" });
    fireEvent.change(input, { target: { value: "cmd2" } });
    fireEvent.keyDown(input, { key: "Enter" });

    fireEvent.keyDown(input, { key: "ArrowUp" });
    expect(input).toHaveValue("cmd2");

    fireEvent.keyDown(input, { key: "ArrowUp" });
    expect(input).toHaveValue("cmd1");

    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(input).toHaveValue("cmd2");

    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(input).toHaveValue("");
  });

  it("does not clear input on ArrowDown with empty history", () => {
    const onSubmit = vi.fn();
    render(<CommandInput onSubmit={onSubmit} disabled={false} />);

    const input = screen.getByPlaceholderText("Enter command...");
    fireEvent.change(input, { target: { value: "typing" } });
    fireEvent.keyDown(input, { key: "ArrowDown" });

    expect(input).toHaveValue("typing");
  });

  it("refocuses input after submit", () => {
    const onSubmit = vi.fn();
    render(<CommandInput onSubmit={onSubmit} disabled={false} />);

    const input = screen.getByPlaceholderText("Enter command...");
    fireEvent.change(input, { target: { value: "ls" } });
    fireEvent.keyDown(input, { key: "Enter" });

    const focusSpy = vi.spyOn(input, "focus");
    rafCallback?.(0);
    expect(focusSpy).toHaveBeenCalledOnce();
  });

  it("disables input when disabled prop is true", () => {
    render(<CommandInput onSubmit={vi.fn()} disabled={true} />);
    expect(screen.getByPlaceholderText("Enter command...")).toBeDisabled();
  });

  it("focuses input when disabled changes from true to false", () => {
    const { rerender } = render(<CommandInput onSubmit={vi.fn()} disabled={true} />);
    const input = screen.getByPlaceholderText("Enter command...");
    const focusSpy = vi.spyOn(input, "focus");

    rerender(<CommandInput onSubmit={vi.fn()} disabled={false} />);
    expect(focusSpy).toHaveBeenCalledOnce();
  });
});
