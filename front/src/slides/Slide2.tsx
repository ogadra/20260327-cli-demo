import type { ReactNode } from "react";
import type { SlideProps } from "../components/SlideView";

/** Placeholder slide displayed as the third page of the presentation. */
export const Slide2 = (_props: SlideProps): ReactNode => (
  <div
    style={{
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      width: "100%",
      height: "100%",
      color: "#fff",
      fontSize: "min(6vw, 48px)",
      fontFamily: "sans-serif",
    }}
  >
    Slide 2
  </div>
);
