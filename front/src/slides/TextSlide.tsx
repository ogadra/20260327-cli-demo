import type { ReactNode } from "react";

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

/** Parse inline markdown formatting and return React nodes. */
const parseInline = (text: string): ReactNode[] => {
  const parts: ReactNode[] = [];
  const regex = /(\*\*(.+?)\*\*|`(.+?)`)/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      parts.push(text.slice(lastIndex, match.index));
    }
    if (match[2] !== undefined) {
      parts.push(<strong key={match.index}>{match[2]}</strong>);
    } else if (match[3] !== undefined) {
      parts.push(
        <code
          key={match.index}
          style={{ background: "rgba(255,255,255,0.1)", padding: "2px 6px", borderRadius: "4px" }}
        >
          {match[3]}
        </code>,
      );
    }
    lastIndex = regex.lastIndex;
  }
  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex));
  }
  return parts;
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
