import type { RefObject } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useExecute } from "./useExecute";
import type { TerminalHandle } from "../components/Terminal";

vi.mock("../api/client", () => ({
  execute: vi.fn(),
  SseEventType: { STDOUT: "stdout", STDERR: "stderr", COMPLETE: "complete" },
}));

import { execute } from "../api/client";

const mockExecute = vi.mocked(execute);

beforeEach(() => {
  mockExecute.mockReset();
});

afterEach(() => {
  vi.restoreAllMocks();
});

/** Mock terminal methods returned alongside the ref for assertion. */
interface MockTerminalRef {
  /** Ref object to pass to useExecute. */
  ref: RefObject<TerminalHandle>;
  /** Mock for TerminalHandle.write. */
  write: ReturnType<typeof vi.fn>;
  /** Mock for TerminalHandle.writeln. */
  writeln: ReturnType<typeof vi.fn>;
}

/** Create a mock TerminalHandle ref for testing. */
const makeTerminalRef = (): MockTerminalRef => {
  const write = vi.fn();
  const writeln = vi.fn();
  return { ref: { current: { write, writeln } }, write, writeln };
};

describe("useExecute", () => {
  it("writes stdout to terminal", async () => {
    const { ref, write, writeln } = makeTerminalRef();

    async function* fakeExecute() {
      yield { type: "stdout" as const, data: "hello\n" };
      yield { type: "complete" as const, exitCode: 0 };
    }
    mockExecute.mockReturnValue(fakeExecute());

    const { result } = renderHook(() => useExecute(true, ref));

    await act(async () => {
      await result.current.run("echo hello");
    });

    expect(writeln).toHaveBeenCalledWith("echo hello");
    expect(write).toHaveBeenCalledWith("hello\n");
    expect(write).toHaveBeenCalledWith("$ ");
  });

  it("writes stderr in red", async () => {
    const { ref, write, writeln } = makeTerminalRef();

    async function* fakeExecute() {
      yield { type: "stderr" as const, data: "err\n" };
      yield { type: "complete" as const, exitCode: 1 };
    }
    mockExecute.mockReturnValue(fakeExecute());

    const { result } = renderHook(() => useExecute(true, ref));

    await act(async () => {
      await result.current.run("bad");
    });

    expect(write).toHaveBeenCalledWith("\x1b[90merr\n\x1b[0m");
    expect(writeln).toHaveBeenCalledWith("\x1b[31mexit code: 1\x1b[0m");
    expect(write).toHaveBeenCalledWith("$ ");
  });

  it("does not run when session is not ready", async () => {
    const { ref } = makeTerminalRef();

    const { result } = renderHook(() => useExecute(false, ref));

    await act(async () => {
      await result.current.run("cmd");
    });

    expect(mockExecute).not.toHaveBeenCalled();
  });

  it("prevents concurrent executions", async () => {
    const { ref } = makeTerminalRef();

    let resolveFirst!: () => void;
    const firstPromise = new Promise<void>((resolve) => {
      resolveFirst = resolve;
    });

    async function* slowExecute() {
      await firstPromise;
      yield { type: "complete" as const, exitCode: 0 };
    }
    mockExecute.mockReturnValue(slowExecute());

    const { result } = renderHook(() => useExecute(true, ref));

    const firstRun = act(async () => {
      await result.current.run("first");
    });

    await act(async () => {
      await result.current.run("second");
    });

    resolveFirst();
    await firstRun;

    expect(mockExecute).toHaveBeenCalledTimes(1);
  });

  it("displays reassignment notification when onReassigned is called", async () => {
    const { ref, writeln } = makeTerminalRef();

    mockExecute.mockImplementation((_command: string, onReassigned?: () => void) => {
      if (onReassigned) onReassigned();
      return (async function* () {
        yield { type: "complete" as const, exitCode: 0 };
      })();
    });

    const { result } = renderHook(() => useExecute(true, ref));

    await act(async () => {
      await result.current.run("ls");
    });

    expect(writeln).toHaveBeenCalledWith(
      "\x1b[33mSession was reassigned. Shell state has been reset.\x1b[0m",
    );
  });

  it("displays error message in terminal on execute failure", async () => {
    const { ref, write, writeln } = makeTerminalRef();

    async function* failingExecute() {
      throw new Error("connection refused");
      yield { type: "complete" as const, exitCode: 0 };
    }
    mockExecute.mockReturnValue(failingExecute());

    const { result } = renderHook(() => useExecute(true, ref));

    await act(async () => {
      await result.current.run("bad-cmd");
    });

    expect(writeln).toHaveBeenCalledWith("\x1b[31mError: connection refused\x1b[0m");
    expect(write).toHaveBeenCalledWith("$ ");
  });

  it("aborts running command on unmount", async () => {
    const { ref } = makeTerminalRef();

    let resolveExecution!: () => void;
    const executionPromise = new Promise<void>((resolve) => {
      resolveExecution = resolve;
    });

    let receivedSignal: AbortSignal | undefined;

    mockExecute.mockImplementation(
      (_command: string, _onReassigned?: () => void, signal?: AbortSignal) => {
        receivedSignal = signal;
        return (async function* () {
          await executionPromise;
          yield { type: "complete" as const, exitCode: 0 };
        })();
      },
    );

    const { result, unmount } = renderHook(() => useExecute(true, ref));

    act(() => {
      void result.current.run("long-running");
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(receivedSignal).toBeDefined();
    expect(receivedSignal!.aborted).toBe(false);

    unmount();

    expect(receivedSignal!.aborted).toBe(true);

    resolveExecution();
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });
  });
});
