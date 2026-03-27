import type { ReactNode } from "react";
import { useRef } from "react";
import Terminal, { type TerminalHandle } from "../components/Terminal";
import CommandInput from "../components/CommandInput";
import { useExecute } from "../hooks/useExecute";

/** Props for the TerminalPane component. */
interface TerminalPaneProps {
  /** Whether the session is ready to accept commands. */
  ready: boolean;
  /** Placeholder text shown in the command input. */
  placeholder: string;
}

/** Single terminal with command input, used as a pane inside TerminalSlide. */
export const TerminalPane = ({ ready, placeholder }: TerminalPaneProps): ReactNode => {
  const terminalRef = useRef<TerminalHandle>(null);
  const { run, running } = useExecute(ready, terminalRef);

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        flex: 1,
        minWidth: 0,
      }}
    >
      <CommandInput onSubmit={run} disabled={!ready || running} placeholder={placeholder} />
      <Terminal ref={terminalRef} />
    </div>
  );
};
