import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import Terminal, { type TerminalHandle } from "./components/Terminal";
import CommandInput from "./components/CommandInput";
import SlideView from "./components/SlideView";
import { MessageType } from "./api/presenter";
import { useSession } from "./hooks/useSession";
import { useExecute } from "./hooks/useExecute";
import { usePresenter } from "./hooks/usePresenter";

/** WebSocket URL derived from the current page origin. */
const wsUrl = (): string => location.origin.replace(/^http/, "ws") + "/ws";

/** Root application component that wires session, terminal, slide view, and command input together. */
const App = (): ReactNode => {
  const ready = useSession();
  const terminalRef = useRef<TerminalHandle>(null);
  const { run, running } = useExecute(ready, terminalRef);
  const { page, mode, instruction, placeholder, viewerCount } = usePresenter(wsUrl());

  useEffect(() => {
    if (ready) {
      terminalRef.current?.write("$ ");
    }
  }, [ready]);

  return (
    <div style={{ position: "relative", height: "100dvh", background: "#000" }}>
      <div
        data-testid="viewer-count"
        style={{
          position: "absolute",
          top: 8,
          right: 8,
          zIndex: 10,
          color: "#aaa",
          fontSize: "12px",
          fontFamily: "sans-serif",
        }}
      >
        {viewerCount} viewers
      </div>

      <div
        data-testid="slide-mode"
        style={{
          display: mode === MessageType.SlideSync ? "flex" : "none",
          flexDirection: "column",
          height: "100%",
        }}
      >
        <div style={{ flexGrow: 1, overflow: "hidden" }}>
          <SlideView page={page} />
        </div>
        {instruction && (
          <div
            data-testid="instruction"
            style={{
              padding: "16px",
              textAlign: "center",
              color: "#888",
              fontSize: "14px",
              fontFamily: "sans-serif",
              borderTop: "1px solid #333",
            }}
          >
            {instruction}
          </div>
        )}
      </div>

      <div
        data-testid="hands-on-mode"
        style={{
          display: mode === MessageType.HandsOn ? "flex" : "none",
          flexDirection: "column",
          height: "100%",
        }}
      >
        {mode === MessageType.HandsOn && (
          <CommandInput
            onSubmit={run}
            disabled={!ready || running}
            placeholder={placeholder || undefined}
          />
        )}
        <Terminal ref={terminalRef} />
      </div>
    </div>
  );
};

export default App;
