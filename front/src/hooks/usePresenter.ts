import { useCallback, useEffect, useRef, useState } from "react";
import { MessageType, parsePresenterMessage, type PresenterMode } from "../api/presenter";

/** Maximum reconnection delay in milliseconds. */
const MAX_DELAY = 8000;

/** Return value of the usePresenter hook. */
export interface UsePresenterResult {
  page: number;
  mode: PresenterMode;
  instruction: string;
  placeholder: string;
  viewerCount: number;
  sendSlideSync: (page: number) => void;
  sendHandsOn: (instruction: string, placeholder: string) => void;
}

/** Dependencies injectable for testing. */
export interface UsePresenterDeps {
  WebSocket: typeof WebSocket;
}

/**
 * Hook that manages a presenter WebSocket connection including page, mode, and viewer count.
 * Connects on mount with auto-reconnect and disconnects on unmount.
 * @param wsUrl - WebSocket URL to connect to.
 * @param deps - Optional dependency overrides for testing.
 * @returns Presenter state and send functions.
 */
export const usePresenter = (
  wsUrl: string,
  deps?: Partial<UsePresenterDeps>,
): UsePresenterResult => {
  const [page, setPage] = useState(0);
  const [mode, setMode] = useState<PresenterMode>(MessageType.SlideSync);
  const [instruction, setInstruction] = useState("");
  const [placeholder, setPlaceholder] = useState("");
  const [viewerCount, setViewerCount] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const WS = deps?.WebSocket ?? globalThis.WebSocket;
    let closed = false;
    let delay = 1000;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const connect = (): void => {
      const ws = new WS(wsUrl);
      wsRef.current = ws;

      ws.onopen = (): void => {
        delay = 1000;
      };

      ws.onmessage = (event: MessageEvent): void => {
        const msg = parsePresenterMessage(String(event.data));
        if (!msg) return;
        switch (msg.type) {
          case MessageType.SlideSync:
            setPage(msg.page);
            setMode(MessageType.SlideSync);
            break;
          case MessageType.HandsOn:
            setInstruction(msg.instruction);
            setPlaceholder(msg.placeholder);
            setMode(MessageType.HandsOn);
            break;
          case MessageType.ViewerCount:
            setViewerCount(msg.count);
            break;
        }
      };

      ws.onclose = (): void => {
        if (!closed) {
          timer = setTimeout(() => {
            timer = null;
            connect();
          }, delay);
          delay = Math.min(delay * 2, MAX_DELAY);
        }
      };
    };

    connect();

    return (): void => {
      closed = true;
      if (timer !== null) clearTimeout(timer);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [wsUrl, deps?.WebSocket]);

  const sendSlideSync = useCallback((p: number): void => {
    wsRef.current?.send(
      JSON.stringify({ action: "message", type: MessageType.SlideSync, page: p }),
    );
  }, []);

  const sendHandsOn = useCallback((inst: string, ph: string): void => {
    wsRef.current?.send(
      JSON.stringify({
        action: "message",
        type: MessageType.HandsOn,
        instruction: inst,
        placeholder: ph,
      }),
    );
  }, []);

  return { page, mode, instruction, placeholder, viewerCount, sendSlideSync, sendHandsOn };
};
