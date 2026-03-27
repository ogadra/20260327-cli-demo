import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import Terminal, { type TerminalHandle } from "../components/Terminal";
import CommandInput from "../components/CommandInput";
import { useExecute } from "../hooks/useExecute";
import type { SessionStatus } from "../hooks/useSession";

/** Props for the TerminalPane component. */
interface TerminalPaneProps {
  /** Current session connection status. */
  sessionStatus: SessionStatus;
  /** Placeholder text shown in the command input. */
  placeholder: string;
  /** Explicit height for the pane. When omitted the pane stretches via flex. */
  height?: string;
}

/** Single terminal with command input, used as a pane inside TerminalSlide. */
export const TerminalPane = ({
  sessionStatus,
  placeholder,
  height,
}: TerminalPaneProps): ReactNode => {
  const ready = sessionStatus === "ready";
  const terminalRef = useRef<TerminalHandle>(null);
  const { run, running } = useExecute(ready, terminalRef);

  /** Write session status messages to the terminal. */
  useEffect(() => {
    const term = terminalRef.current;
    if (!term) return;
    if (sessionStatus === "loading") {
      term.writeln("Connecting to runner...");
    } else if (sessionStatus === "retrying") {
      term.writeln("\x1b[33mConnection failed. Retrying...\x1b[0m");
    }
  }, [sessionStatus]);

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        height,
        flex: height ? undefined : 1,
        flexShrink: height ? 0 : undefined,
        minWidth: 0,
      }}
    >
      <CommandInput onSubmit={run} disabled={!ready || running} placeholder={placeholder} />
      <Terminal ref={terminalRef} />
    </div>
  );
};
