import type { ReactNode } from "react";
import { parseInline } from "./parseInline";

/** Props for the TextSlide component. */
interface TextSlideProps {
  /** Lines of text to display, rendered as separate paragraphs. */
  lines: string[];
}

/** Compute total character count across all lines for font size selection. */
const totalLength = (lines: string[]): number => lines.reduce((sum, line) => sum + line.length, 0);

/** Select font size based on total text length. */
const fontSize = (lines: string[]): string => {
  const len = totalLength(lines);
  if (len <= 20) return "min(8vw, 64px)";
  if (len <= 50) return "min(5vw, 40px)";
  return "min(3.5vw, 28px)";
};

/** Slide component that displays centered text with auto-sized font. */
export const TextSlide = ({ lines }: TextSlideProps): ReactNode => (
  <div
    style={{
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      flexDirection: "column",
      width: "100%",
      height: "100%",
      color: "#fff",
      fontSize: fontSize(lines),
      fontFamily: "sans-serif",
      textAlign: "center",
      padding: "32px",
      boxSizing: "border-box",
    }}
  >
    {lines.map((line, i) => (
      <div key={i} style={{ marginBottom: i < lines.length - 1 ? "0.5em" : 0 }}>
        {parseInline(line)}
      </div>
    ))}
  </div>
);
