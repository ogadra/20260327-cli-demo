import { useCallback, useEffect, useRef, useState } from "react";
import {
  Action,
  ClientMessageType,
  ServerMessageType,
  parsePresenterMessage,
  type PresenterMode,
} from "../api/presenter";

/** Maximum reconnection delay in milliseconds. */
const MAX_DELAY = 8000;

/** Snapshot of a single poll state received from the server. */
export interface PollStateData {
  options: string[];
  maxChoices: number;
  votes: Record<string, number>;
  myChoices: string[];
}

/** Return value of the usePresenter hook. */
export interface UsePresenterResult {
  page: number;
  mode: PresenterMode;
  instruction: string;
  placeholder: string;
  viewerCount: number;
  pollStates: Partial<Record<string, PollStateData>>;
  sendSlideSync: (page: number) => void;
  sendHandsOn: (instruction: string, placeholder: string) => void;
  sendPollOpen: (pollId: string, options: string[], maxChoices: number) => void;
  sendPollGet: (pollId: string, options: string[], maxChoices: number) => void;
  sendPollVote: (pollId: string, choice: string) => void;
  sendPollUnvote: (pollId: string, choice: string) => void;
  sendPollSwitch: (pollId: string, from: string, to: string) => void;
}

/** Dependencies injectable for testing. */
export interface UsePresenterDeps {
  WebSocket: typeof WebSocket;
}

/**
 * Hook that manages a presenter WebSocket connection including page, mode, viewer count, and poll states keyed by pollId.
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
  const [mode, setMode] = useState<PresenterMode>(ServerMessageType.SlideSync);
  const [instruction, setInstruction] = useState("");
  const [placeholder, setPlaceholder] = useState("");
  const [viewerCount, setViewerCount] = useState(0);
  const [pollStates, setPollStates] = useState<Record<string, PollStateData>>({});
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
        ws.send(JSON.stringify({ action: "message", type: "get_state" }));
      };

      ws.onmessage = (event: MessageEvent): void => {
        const msg = parsePresenterMessage(String(event.data));
        if (!msg) return;
        switch (msg.type) {
          case ServerMessageType.SlideSync:
            setPage(msg.page);
            setMode(ServerMessageType.SlideSync);
            break;
          case ServerMessageType.HandsOn:
            setInstruction(msg.instruction);
            setPlaceholder(msg.placeholder);
            setMode(ServerMessageType.HandsOn);
            break;
          case ServerMessageType.ViewerCount:
            setViewerCount(msg.count);
            break;
          case ServerMessageType.PollState:
            setPollStates((prev) => ({
              ...prev,
              [msg.pollId]: {
                options: msg.options,
                maxChoices: msg.maxChoices,
                votes: msg.votes,
                myChoices: msg.myChoices,
              },
            }));
            break;
          case ServerMessageType.PollError:
            setPollStates((prev) => {
              const existing = prev[msg.pollId];
              if (!existing) return prev;
              return {
                ...prev,
                [msg.pollId]: { ...existing, votes: msg.votes, myChoices: msg.myChoices },
              };
            });
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
    wsRef.current?.send(JSON.stringify({ action: "message", type: Action.SlideSync, page: p }));
  }, []);

  const sendHandsOn = useCallback((inst: string, ph: string): void => {
    wsRef.current?.send(
      JSON.stringify({
        action: "message",
        type: Action.HandsOn,
        instruction: inst,
        placeholder: ph,
      }),
    );
  }, []);

  /** Send a poll_open message to start a poll and broadcast it to all viewers. */
  const sendPollOpen = useCallback(
    (pollId: string, options: string[], maxChoices: number): void => {
      wsRef.current?.send(
        JSON.stringify({
          action: "message",
          type: ClientMessageType.PollOpen,
          pollId,
          options,
          maxChoices,
        }),
      );
    },
    [],
  );

  /** Send a poll_get message to initialize or retrieve a poll. */
  const sendPollGet = useCallback((pollId: string, options: string[], maxChoices: number): void => {
    wsRef.current?.send(
      JSON.stringify({
        action: "message",
        type: ClientMessageType.PollGet,
        pollId,
        options,
        maxChoices,
      }),
    );
  }, []);

  /** Send a poll_vote message to cast a vote. */
  const sendPollVote = useCallback((pollId: string, choice: string): void => {
    wsRef.current?.send(
      JSON.stringify({ action: "message", type: ClientMessageType.PollVote, pollId, choice }),
    );
  }, []);

  /** Send a poll_unvote message to withdraw a vote. */
  const sendPollUnvote = useCallback((pollId: string, choice: string): void => {
    wsRef.current?.send(
      JSON.stringify({ action: "message", type: ClientMessageType.PollUnvote, pollId, choice }),
    );
  }, []);

  /** Send a poll_switch message to change a vote from one option to another. */
  const sendPollSwitch = useCallback((pollId: string, from: string, to: string): void => {
    wsRef.current?.send(
      JSON.stringify({ action: "message", type: ClientMessageType.PollSwitch, pollId, from, to }),
    );
  }, []);

  return {
    page,
    mode,
    instruction,
    placeholder,
    viewerCount,
    pollStates,
    sendSlideSync,
    sendHandsOn,
    sendPollOpen,
    sendPollGet,
    sendPollVote,
    sendPollUnvote,
    sendPollSwitch,
  };
};
