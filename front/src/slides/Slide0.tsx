import type { ReactNode } from "react";
import type { SlideProps } from "../components/SlideView";

/** Title slide displayed as the first page of the presentation. */
const Slide0 = (_props: SlideProps): ReactNode => (
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
    Welcome
  </div>
);

export default Slide0;
