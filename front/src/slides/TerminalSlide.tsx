import type { ReactNode } from "react";
import type { SessionStatus } from "../hooks/useSession";
import { TerminalPane } from "./TerminalPane";

/** Props for the TerminalSlide component. */
interface TerminalSlideProps {
  /** Current session connection status. */
  sessionStatus: SessionStatus;
  /** Instructional text displayed above the terminals. */
  instruction: string;
  /** Commands shown as placeholder hints, one per terminal pane. */
  commands: string[];
}

/** Slide component with one or more terminal panes arranged side by side. */
export const TerminalSlide = ({
  sessionStatus,
  instruction,
  commands,
}: TerminalSlideProps): ReactNode => {
  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        width: "100%",
        height: "100%",
        background: "#000",
      }}
    >
      {instruction && (
        <div
          style={{
            padding: "16px",
            color: "#fff",
            fontSize: "min(4vw, 20px)",
            fontFamily: "sans-serif",
            textAlign: "center",
          }}
        >
          {instruction}
        </div>
      )}
      <div
        style={{
          display: "flex",
          flex: 1,
          gap: "4px",
          minHeight: 0,
        }}
      >
        {commands.map((cmd, index) => (
          <TerminalPane key={`${index}-${cmd}`} sessionStatus={sessionStatus} placeholder={cmd} />
        ))}
      </div>
    </div>
  );
};
