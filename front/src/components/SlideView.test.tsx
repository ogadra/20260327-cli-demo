import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import SlideView from "./SlideView";

vi.mock("../slides/index", () => {
  return {
    default: [() => <div>Slide Zero</div>, () => <div>Slide One</div>],
  };
});

describe("SlideView", () => {
  it("renders the slide for page 0", () => {
    render(<SlideView page={0} />);
    expect(screen.getByText("Slide Zero")).toBeDefined();
    expect(screen.getByTestId("slide-content")).toBeDefined();
  });

  it("renders the slide for page 1", () => {
    render(<SlideView page={1} />);
    expect(screen.getByText("Slide One")).toBeDefined();
  });

  it("renders fallback for out-of-range page", () => {
    render(<SlideView page={99} />);
    expect(screen.getByText("Slide not found")).toBeDefined();
    expect(screen.getByTestId("slide-fallback")).toBeDefined();
  });

  it("renders fallback for negative page", () => {
    render(<SlideView page={-1} />);
    expect(screen.getByText("Slide not found")).toBeDefined();
  });
});
