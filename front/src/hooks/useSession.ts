import { useEffect, useRef, useState } from "react";
import { createSession, deleteSession } from "../api/client";

/**
 * Hook that creates a session on mount and attempts to delete it on unmount.
 * Returns whether the session is ready.
 */
export const useSession = (): boolean => {
  const [ready, setReady] = useState(false);
  const readyRef = useRef(false);

  useEffect(() => {
    const ac = new AbortController();

    void (async () => {
      try {
        await createSession(ac.signal);
        if (ac.signal.aborted) return;
        readyRef.current = true;
        setReady(true);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        if (ac.signal.aborted) return;
        console.error("Failed to create session", err);
      }
    })();

    return () => {
      ac.abort();
      if (readyRef.current) {
        deleteSession();
      }
    };
  }, []);

  return ready;
};
