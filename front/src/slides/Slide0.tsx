import type { ReactNode } from "react";
import { useRef } from "react";
import type { SlideProps } from "../components/SlideView";
import Terminal, { type TerminalHandle } from "../components/Terminal";
import CommandInput from "../components/CommandInput";
import { useSession } from "../hooks/useSession";
import { useExecute } from "../hooks/useExecute";

/** Hands-on slide with embedded terminal and command input. */
export const Slide0 = (_props: SlideProps): ReactNode => {
  const ready = useSession();
  const terminalRef = useRef<TerminalHandle>(null);
  const { run, running } = useExecute(ready, terminalRef);

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
      <div
        style={{
          padding: "16px",
          color: "#fff",
          fontSize: "min(4vw, 20px)",
          fontFamily: "sans-serif",
          textAlign: "center",
        }}
      >
        Try running a command
      </div>
      <CommandInput onSubmit={run} disabled={!ready || running} placeholder="echo hello" />
      <Terminal ref={terminalRef} />
    </div>
  );
};
