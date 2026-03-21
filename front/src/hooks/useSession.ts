import { useEffect, useRef, useState } from "react";
import { createSession, deleteSession } from "../api/client";

export const useSession = (): string | null => {
  const [sessionId, setSessionId] = useState<string | null>(null);
  const sessionIdRef = useRef<string | null>(null);

  useEffect(() => {
    const ac = new AbortController();

    void (async () => {
      const res = await createSession();
      if (ac.signal.aborted) return;
      sessionIdRef.current = res.sessionId;
      setSessionId(res.sessionId);
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
