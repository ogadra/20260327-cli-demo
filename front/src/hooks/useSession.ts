import { useEffect, useRef, useState } from "react";
import { createSession, deleteSession } from "../api/client";

/** Possible states of the session lifecycle. */
export type SessionStatus = "loading" | "ready" | "retrying";

/** Maximum retry delay in milliseconds. */
const MAX_DELAY_MS = 8000;

/**
 * Hook that creates a session on mount with automatic retry on failure.
 * Returns the current session status. Cleans up the session on unmount.
 */
export const useSession = (): SessionStatus => {
  const [status, setStatus] = useState<SessionStatus>("loading");
  const readyRef = useRef(false);

  useEffect(() => {
    const ac = new AbortController();
    let retryTimeout: ReturnType<typeof setTimeout> | undefined;

    const attempt = async (delay: number): Promise<void> => {
      try {
        await createSession(ac.signal);
        if (ac.signal.aborted) return;
        readyRef.current = true;
        setStatus("ready");
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        if (ac.signal.aborted) return;
        console.error("Failed to create session", err);
        setStatus("retrying");
        const nextDelay = Math.min(delay * 2, MAX_DELAY_MS);
        retryTimeout = setTimeout(() => {
          if (!ac.signal.aborted) {
            void attempt(nextDelay);
          }
        }, delay);
      }
    };

    void attempt(1000);

    return () => {
      ac.abort();
      if (retryTimeout !== undefined) {
        clearTimeout(retryTimeout);
      }
      if (readyRef.current) {
        deleteSession();
      }
    };
  }, []);

  return status;
};
