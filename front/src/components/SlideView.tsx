import type { ReactNode } from "react";
import slides from "../slides/index";

/** Props for the SlideView component. */
interface SlideViewProps {
  page: number;
}

/** Renders the slide component corresponding to the given page number. */
const SlideView = ({ page }: SlideViewProps): ReactNode => {
  const Slide = slides[page];
  if (!Slide) {
    return (
      <div
        data-testid="slide-fallback"
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          width: "100%",
          height: "100%",
          color: "#888",
          fontSize: "min(4vw, 24px)",
          fontFamily: "sans-serif",
        }}
      >
        Slide not found
      </div>
    );
  }
  return (
    <div data-testid="slide-content" style={{ width: "100%", height: "100%" }}>
      <Slide />
    </div>
  );
};

export default SlideView;
