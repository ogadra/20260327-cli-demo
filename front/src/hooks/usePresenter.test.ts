import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { usePresenter } from "./usePresenter";
import { Action, ClientMessageType, ServerMessageType } from "../api/presenter";

/** Minimal mock WebSocket instances tracker. */
const instances: MockWebSocket[] = [];

/** Minimal mock of the WebSocket interface for testing. */
function MockWebSocket(this: MockWebSocket, url: string): void {
  this.url = url;
  this.sent = [];
  this.closeCalled = false;
  this.onopen = null;
  this.onmessage = null;
  this.onclose = null;
  instances.push(this);
}

MockWebSocket.prototype.send = function (data: string): void {
  (this as MockWebSocket).sent.push(data);
};

MockWebSocket.prototype.close = function (): void {
  (this as MockWebSocket).closeCalled = true;
};

interface MockWebSocket {
  url: string;
  sent: string[];
  closeCalled: boolean;
  onopen: (() => void) | null;
  onmessage: ((event: { data: string }) => void) | null;
  onclose: (() => void) | null;
}

/** Return the latest MockWebSocket instance. */
const latest = (): MockWebSocket => {
  const inst = instances.at(-1);
  if (!inst) throw new Error("No MockWebSocket instance found");
  return inst;
};

/** Simulate the open event on the latest instance. */
const simulateOpen = (): void => {
  latest().onopen?.();
};

/** Simulate a message event with the given data string on the latest instance. */
const simulateMessage = (data: string): void => {
  latest().onmessage?.({ data });
};

/** Simulate the close event on the latest instance. */
const simulateClose = (): void => {
  latest().onclose?.();
};

/** Render the hook with mock WebSocket injected. */
// eslint-disable-next-line @typescript-eslint/explicit-function-return-type
const renderPresenter = () =>
  renderHook(() =>
    usePresenter("ws://test", { WebSocket: MockWebSocket as unknown as typeof WebSocket }),
  );

beforeEach(() => {
  vi.useFakeTimers();
  instances.length = 0;
});

afterEach(() => {
  vi.useRealTimers();
});

describe("usePresenter", () => {
  it("connects to the given URL", () => {
    renderPresenter();
    expect(latest().url).toBe("ws://test");
  });

  it("returns initial state", () => {
    const { result } = renderPresenter();
    expect(result.current.page).toBe(0);
    expect(result.current.mode).toBe(ServerMessageType.SlideSync);
    expect(result.current.instruction).toBe("");
    expect(result.current.placeholder).toBe("");
    expect(result.current.viewerCount).toBe(0);
  });

  it("updates page and sets mode to slide_sync on slide_sync message", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage(JSON.stringify({ type: ServerMessageType.SlideSync, page: 5 }));
    });
    expect(result.current.page).toBe(5);
    expect(result.current.mode).toBe(ServerMessageType.SlideSync);
  });

  it("updates instruction, placeholder, and sets mode to hands_on on hands_on message", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage(
        JSON.stringify({
          type: ServerMessageType.HandsOn,
          instruction: "run echo",
          placeholder: "$ echo hi",
        }),
      );
    });
    expect(result.current.mode).toBe(ServerMessageType.HandsOn);
    expect(result.current.instruction).toBe("run echo");
    expect(result.current.placeholder).toBe("$ echo hi");
  });

  it("updates viewerCount on viewer_count message", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage(JSON.stringify({ type: ServerMessageType.ViewerCount, count: 42 }));
    });
    expect(result.current.viewerCount).toBe(42);
  });

  it("switches from hands_on back to slide_sync on slide_sync", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage(
        JSON.stringify({
          type: ServerMessageType.HandsOn,
          instruction: "do something",
          placeholder: "$ ...",
        }),
      );
    });
    expect(result.current.mode).toBe(ServerMessageType.HandsOn);

    act(() => {
      simulateMessage(JSON.stringify({ type: ServerMessageType.SlideSync, page: 3 }));
    });
    expect(result.current.mode).toBe(ServerMessageType.SlideSync);
    expect(result.current.page).toBe(3);
  });

  it("ignores unknown message types", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage(JSON.stringify({ type: "unknown" }));
    });
    expect(result.current.page).toBe(0);
    expect(result.current.mode).toBe(ServerMessageType.SlideSync);
  });

  it("ignores malformed JSON messages", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage("not json");
    });
    expect(result.current.page).toBe(0);
  });

  it("sends slide_sync message via sendSlideSync", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      result.current.sendSlideSync(7);
    });
    expect(latest().sent).toEqual([
      JSON.stringify({ action: "message", type: Action.SlideSync, page: 7 }),
    ]);
  });

  it("sends hands_on message via sendHandsOn", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      result.current.sendHandsOn("try this", "$ try");
    });
    expect(latest().sent).toEqual([
      JSON.stringify({
        action: "message",
        type: Action.HandsOn,
        instruction: "try this",
        placeholder: "$ try",
      }),
    ]);
  });

  it("does not throw sendSlideSync after unmount", () => {
    const { result, unmount } = renderPresenter();
    unmount();
    expect(() => result.current.sendSlideSync(1)).not.toThrow();
  });

  it("does not throw sendHandsOn after unmount", () => {
    const { result, unmount } = renderPresenter();
    unmount();
    expect(() => result.current.sendHandsOn("test", "ph")).not.toThrow();
  });

  it("closes WebSocket on unmount", () => {
    const { unmount } = renderPresenter();
    simulateOpen();
    unmount();
    expect(latest().closeCalled).toBe(true);
  });

  it("reconnects with exponential backoff after close", () => {
    renderPresenter();
    simulateOpen();
    act(() => {
      simulateClose();
    });

    expect(instances).toHaveLength(1);
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(instances).toHaveLength(2);

    act(() => {
      simulateClose();
    });
    act(() => {
      vi.advanceTimersByTime(1999);
    });
    expect(instances).toHaveLength(2);
    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(instances).toHaveLength(3);
  });

  it("caps reconnection delay at 8 seconds", () => {
    renderPresenter();

    for (let i = 0; i < 5; i++) {
      act(() => {
        simulateClose();
      });
      act(() => {
        vi.advanceTimersByTime(8000);
      });
    }

    const countBefore = instances.length;
    act(() => {
      simulateClose();
    });

    act(() => {
      vi.advanceTimersByTime(7999);
    });
    expect(instances).toHaveLength(countBefore);

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(instances).toHaveLength(countBefore + 1);
  });

  it("resets delay after successful open", () => {
    renderPresenter();
    simulateOpen();
    act(() => {
      simulateClose();
    });

    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(instances).toHaveLength(2);

    simulateOpen();
    act(() => {
      simulateClose();
    });

    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(instances).toHaveLength(3);
  });

  it("does not reconnect after unmount", () => {
    const { unmount } = renderPresenter();
    simulateOpen();
    unmount();
    act(() => {
      simulateClose();
    });

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(instances).toHaveLength(1);
  });

  it("clears pending reconnect timer on unmount", () => {
    const { unmount } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateClose();
    });

    unmount();
    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(instances).toHaveLength(1);
  });

  it("returns initial pollStates as empty object", () => {
    const { result } = renderPresenter();
    expect(result.current.pollStates).toEqual({});
  });

  it("updates pollStates on poll_state message keyed by pollId", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage(
        JSON.stringify({
          type: ServerMessageType.PollState,
          pollId: "q1",
          options: ["A", "B"],
          maxChoices: 1,
          votes: { A: 10, B: 5 },
          myChoices: ["A"],
        }),
      );
    });
    expect(result.current.pollStates["q1"]).toEqual({
      options: ["A", "B"],
      maxChoices: 1,
      votes: { A: 10, B: 5 },
      myChoices: ["A"],
    });
  });

  it("stores multiple polls independently", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage(
        JSON.stringify({
          type: ServerMessageType.PollState,
          pollId: "q1",
          options: ["A", "B"],
          maxChoices: 1,
          votes: { A: 1 },
          myChoices: [],
        }),
      );
    });
    act(() => {
      simulateMessage(
        JSON.stringify({
          type: ServerMessageType.PollState,
          pollId: "q2",
          options: ["X", "Y", "Z"],
          maxChoices: 2,
          votes: { X: 3 },
          myChoices: ["X"],
        }),
      );
    });
    expect(result.current.pollStates["q1"]).toEqual({
      options: ["A", "B"],
      maxChoices: 1,
      votes: { A: 1 },
      myChoices: [],
    });
    expect(result.current.pollStates["q2"]).toEqual({
      options: ["X", "Y", "Z"],
      maxChoices: 2,
      votes: { X: 3 },
      myChoices: ["X"],
    });
  });

  it("does not change mode on poll_state message", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage(
        JSON.stringify({
          type: ServerMessageType.PollState,
          pollId: "q1",
          options: ["A"],
          maxChoices: 1,
          votes: {},
          myChoices: [],
        }),
      );
    });
    expect(result.current.mode).toBe(ServerMessageType.SlideSync);
  });

  it("updates pollStates votes and myChoices on poll_error message", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage(
        JSON.stringify({
          type: ServerMessageType.PollState,
          pollId: "q1",
          options: ["A", "B"],
          maxChoices: 1,
          votes: { A: 5 },
          myChoices: [],
        }),
      );
    });
    act(() => {
      simulateMessage(
        JSON.stringify({
          type: ServerMessageType.PollError,
          pollId: "q1",
          error: "duplicate vote",
          votes: { A: 6 },
          myChoices: ["A"],
        }),
      );
    });
    expect(result.current.pollStates["q1"]?.votes).toEqual({ A: 6 });
    expect(result.current.pollStates["q1"]?.myChoices).toEqual(["A"]);
  });

  it("ignores poll_error when no prior pollState exists for the pollId", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      simulateMessage(
        JSON.stringify({
          type: ServerMessageType.PollError,
          pollId: "q1",
          error: "not found",
          votes: {},
          myChoices: [],
        }),
      );
    });
    expect(result.current.pollStates["q1"]).toBeUndefined();
  });

  it("sends poll_get message via sendPollGet", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      result.current.sendPollGet("q1", ["A", "B"], 1);
    });
    expect(latest().sent).toEqual([
      JSON.stringify({
        action: "message",
        type: ClientMessageType.PollGet,
        pollId: "q1",
        options: ["A", "B"],
        maxChoices: 1,
      }),
    ]);
  });

  it("sends poll_vote message via sendPollVote", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      result.current.sendPollVote("q1", "A");
    });
    expect(latest().sent).toEqual([
      JSON.stringify({
        action: "message",
        type: ClientMessageType.PollVote,
        pollId: "q1",
        choice: "A",
      }),
    ]);
  });

  it("sends poll_unvote message via sendPollUnvote", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      result.current.sendPollUnvote("q1", "A");
    });
    expect(latest().sent).toEqual([
      JSON.stringify({
        action: "message",
        type: ClientMessageType.PollUnvote,
        pollId: "q1",
        choice: "A",
      }),
    ]);
  });

  it("sends poll_switch message via sendPollSwitch", () => {
    const { result } = renderPresenter();
    simulateOpen();
    act(() => {
      result.current.sendPollSwitch("q1", "A", "B");
    });
    expect(latest().sent).toEqual([
      JSON.stringify({
        action: "message",
        type: ClientMessageType.PollSwitch,
        pollId: "q1",
        from: "A",
        to: "B",
      }),
    ]);
  });

  it("does not throw poll send functions after unmount", () => {
    const { result, unmount } = renderPresenter();
    unmount();
    expect(() => result.current.sendPollGet("q1", ["A"], 1)).not.toThrow();
    expect(() => result.current.sendPollVote("q1", "A")).not.toThrow();
    expect(() => result.current.sendPollUnvote("q1", "A")).not.toThrow();
    expect(() => result.current.sendPollSwitch("q1", "A", "B")).not.toThrow();
  });
});
