import type { RefObject } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
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
 * @param ready - Whether the session is ready to accept commands.
 * @param terminalRef - Ref to the Terminal component for writing output.
 */
export const useExecute = (
  ready: boolean,
  terminalRef: RefObject<TerminalHandle | null>,
): UseExecuteResult => {
  const [running, setRunning] = useState(false);
  const runningRef = useRef(false);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  const run = useCallback(
    async (command: string) => {
      if (!ready || runningRef.current) return;
      const controller = new AbortController();
      abortRef.current = controller;
      runningRef.current = true;
      setRunning(true);

      terminalRef.current?.writeln(command);

      try {
        const onReassigned = (): void => {
          terminalRef.current?.writeln(
            "\x1b[33mSession was reassigned. Shell state has been reset.\x1b[0m",
          );
        };
        for await (const event of execute(command, onReassigned, controller.signal)) {
          handleEvent(event, terminalRef);
        }
      } catch (error) {
        if (controller.signal.aborted) return;
        const message = error instanceof Error ? error.message : "Unknown error";
        terminalRef.current?.writeln(`\x1b[31mError: ${message}\x1b[0m`);
      } finally {
        if (abortRef.current === controller) abortRef.current = null;
        terminalRef.current?.write("$ ");
        runningRef.current = false;
        setRunning(false);
      }
    },
    [ready, terminalRef],
  );

  return { run, running };
};

/** Write an SSE event to the terminal, colouring stderr gray and non-zero exit codes red. */
const handleEvent = (event: SseEvent, terminalRef: RefObject<TerminalHandle | null>): void => {
  switch (event.type) {
    case SseEventType.STDOUT:
      terminalRef.current?.write(event.data);
      break;
    case SseEventType.STDERR:
      terminalRef.current?.write(`\x1b[38;5;252m${event.data}\x1b[0m`);
      break;
    case SseEventType.COMPLETE:
      if (event.exitCode !== 0) {
        terminalRef.current?.writeln(`\x1b[31mexit code: ${event.exitCode}\x1b[0m`);
      }
      break;
  }
};
