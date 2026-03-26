import type { ReactNode } from "react";
import type { SlideProps } from "../components/SlideView";

/** Placeholder slide displayed as the second page of the presentation. */
const Slide1 = (_props: SlideProps): ReactNode => (
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
    Slide 1
  </div>
);

export default Slide1;
