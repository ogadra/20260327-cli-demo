import type { ReactNode } from "react";

/** Parse inline markdown formatting and return React nodes. Newlines are rendered as br elements. */
export const parseInline = (text: string): ReactNode[] => {
  const parts: ReactNode[] = [];
  const regex = /(\*\*(.+?)\*\*|`(.+?)`)/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      const mid = text.slice(lastIndex, match.index);
      mid.split("\n").forEach((seg, i) => {
        if (i > 0) parts.push(<br key={`br-${lastIndex}-${i}`} />);
        if (seg) parts.push(seg);
      });
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
    const tail = text.slice(lastIndex);
    tail.split("\n").forEach((seg, i) => {
      if (i > 0) parts.push(<br key={`br-${lastIndex}-${i}`} />);
      if (seg) parts.push(seg);
    });
  }
  return parts;
};
