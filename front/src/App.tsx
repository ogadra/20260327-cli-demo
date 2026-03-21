import React, { useRef } from "react";
import Terminal, { type TerminalHandle } from "./components/Terminal";
import CommandInput from "./components/CommandInput";
import { useSession } from "./hooks/useSession";
import { useExecute } from "./hooks/useExecute";

/** Root application component that wires session, terminal, and command input together. */
const App = (): React.JSX.Element => {
  const sessionId = useSession();
  const terminalRef = useRef<TerminalHandle>(null);
  const { run, running } = useExecute(sessionId, terminalRef);

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100vh" }}>
      <Terminal ref={terminalRef} />
      <CommandInput onSubmit={run} disabled={!sessionId || running} />
    </div>
  );
};

export default App;
