import type { ReactNode } from "react";

/** Props for the TitleSlide component. */
interface TitleSlideProps {
  /** Title text to display. */
  text: string;
}

/** Slide component that displays a large centered title. */
export const TitleSlide = ({ text }: TitleSlideProps): ReactNode => (
  <div
    style={{
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      width: "100%",
      height: "100%",
      color: "#fff",
      fontSize: "min(10vw, 80px)",
      fontFamily: "sans-serif",
      fontWeight: "bold",
      textAlign: "center",
      padding: "32px",
      boxSizing: "border-box",
    }}
  >
    {text.split("\n").flatMap((part, i) => (i === 0 ? [part] : [<br key={i} />, part]))}
  </div>
);
