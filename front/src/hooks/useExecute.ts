import { useCallback, useRef, useState } from "react";
import { execute, SseEventType, type SseEvent } from "../api/client";
import type { TerminalHandle } from "../components/Terminal";

/** Return type of the useExecute hook. */
interface UseExecuteResult {
  /** Execute a command in the session and stream output to the terminal. */
  run: (command: string) => Promise<void>;
  /** Whether a command is currently executing. */
  running: boolean;
}

/**
 * Hook that executes a command via SSE and streams output to a Terminal ref.
 * @param sessionId - The active session ID, or null if not connected.
 * @param terminalRef - Ref to the Terminal component for writing output.
 */
export const useExecute = (
  sessionId: string | null,
  terminalRef: React.RefObject<TerminalHandle | null>,
): UseExecuteResult => {
  const [running, setRunning] = useState(false);
  const runningRef = useRef(false);

  const run = useCallback(
    async (command: string) => {
      if (!sessionId || runningRef.current) return;
      runningRef.current = true;
      setRunning(true);

      terminalRef.current?.writeln(`$ ${command}`);

      try {
        for await (const event of execute(sessionId, command)) {
          handleEvent(event, terminalRef);
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : "Unknown error";
        terminalRef.current?.writeln(`\x1b[31mError: ${message}\x1b[0m`);
      } finally {
        runningRef.current = false;
        setRunning(false);
      }
    },
    [sessionId, terminalRef],
  );

  return { run, running };
};

/** Write an SSE event to the terminal, colouring stderr and non-zero exit codes red. */
const handleEvent = (
  event: SseEvent,
  terminalRef: React.RefObject<TerminalHandle | null>,
): void => {
  switch (event.type) {
    case SseEventType.STDOUT:
      terminalRef.current?.write(event.data);
      break;
    case SseEventType.STDERR:
      terminalRef.current?.write(`\x1b[31m${event.data}\x1b[0m`);
      break;
    case SseEventType.COMPLETE:
      if (event.exitCode !== 0) {
        terminalRef.current?.writeln(`\x1b[31mexit code: ${event.exitCode}\x1b[0m`);
      }
      break;
  }
};
