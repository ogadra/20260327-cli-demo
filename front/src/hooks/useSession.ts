import { useEffect, useRef, useState } from "react";
import { createSession, deleteSession } from "../api/client";

/**
 * Hook that creates a session on mount and attempts to delete it on unmount.
 * Returns the session ID once available, or null while loading.
 */
export const useSession = (): string | null => {
  const [sessionId, setSessionId] = useState<string | null>(null);
  const sessionIdRef = useRef<string | null>(null);

  useEffect(() => {
    const ac = new AbortController();

    void (async () => {
      try {
        const res = await createSession(ac.signal);
        if (ac.signal.aborted) return;
        sessionIdRef.current = res.sessionId;
        setSessionId(res.sessionId);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        if (ac.signal.aborted) return;
        console.error("Failed to create session", err);
      }
    })();

    return () => {
      ac.abort();
      if (sessionIdRef.current) {
        deleteSession();
      }
    };
  }, []);

  return sessionId;
};
