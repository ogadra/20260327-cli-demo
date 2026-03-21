import { useEffect, useRef, useState } from "react";
import { createSession, deleteSession } from "../api/client";

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
        deleteSession(sessionIdRef.current);
      }
    };
  }, []);

  return sessionId;
};
