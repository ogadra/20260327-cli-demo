import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import Terminal, { type TerminalHandle } from "../components/Terminal";
import CommandInput from "../components/CommandInput";
import { useExecute } from "../hooks/useExecute";
import type { SessionStatus } from "../hooks/useSession";

/** Props for the TerminalSlide component. */
interface TerminalSlideProps {
  /** Current session connection status. */
  sessionStatus: SessionStatus;
  /** Instructional text displayed above the terminal. */
  instruction: string;
  /** Commands shown as placeholder hints in the input field. */
  commands: string[];
}

/** Slide component with an embedded terminal and command input. */
export const TerminalSlide = ({
  sessionStatus,
  instruction,
  commands,
}: TerminalSlideProps): ReactNode => {
  const ready = sessionStatus === "ready";
  const terminalRef = useRef<TerminalHandle>(null);
  const { run, running } = useExecute(ready, terminalRef);
  const placeholder = commands[0] ?? "";

  /** Write session status messages to the terminal. */
  useEffect(() => {
    const term = terminalRef.current;
    if (!term) return;
    if (sessionStatus === "loading") {
      term.writeln("Connecting to runner...");
    } else if (sessionStatus === "retrying") {
      term.writeln("\x1b[33mConnection failed. Retrying...\x1b[0m");
    } else if (sessionStatus === "ready") {
      term.write("$ ");
    }
  }, [sessionStatus]);

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
      <CommandInput onSubmit={run} disabled={!ready || running} placeholder={placeholder} />
      <Terminal ref={terminalRef} />
    </div>
  );
};
