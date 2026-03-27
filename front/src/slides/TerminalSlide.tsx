import type { ReactNode } from "react";
import { useRef } from "react";
import Terminal, { type TerminalHandle } from "../components/Terminal";
import CommandInput from "../components/CommandInput";
import { useSession } from "../hooks/useSession";
import { useExecute } from "../hooks/useExecute";

/** Props for the TerminalSlide component. */
interface TerminalSlideProps {
  /** Instructional text displayed above the terminal. */
  instruction: string;
  /** Commands shown as placeholder hints in the input field. */
  commands: string[];
}

/** Slide component with an embedded terminal and command input. */
export const TerminalSlide = ({ instruction, commands }: TerminalSlideProps): ReactNode => {
  const ready = useSession();
  const terminalRef = useRef<TerminalHandle>(null);
  const { run, running } = useExecute(ready, terminalRef);
  const placeholder = commands[0] ?? "";

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
