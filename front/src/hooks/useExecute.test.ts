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

/** Create a mock TerminalHandle ref for testing. */
function makeTerminalRef(): RefObject<TerminalHandle> {
  return { current: { write: vi.fn(), writeln: vi.fn() } };
}

describe("useExecute", () => {
  it("writes stdout to terminal", async () => {
    const termRef = makeTerminalRef();

    async function* fakeExecute() {
      yield { type: "stdout" as const, data: "hello\n" };
      yield { type: "complete" as const, exitCode: 0 };
    }
    mockExecute.mockReturnValue(fakeExecute());

    const { result } = renderHook(() => useExecute("sess1", termRef));

    await act(async () => {
      await result.current.run("echo hello");
    });

    expect(termRef.current.writeln).toHaveBeenCalledWith("$ echo hello");
    expect(termRef.current.write).toHaveBeenCalledWith("hello\n");
  });

  it("writes stderr in red", async () => {
    const termRef = makeTerminalRef();

    async function* fakeExecute() {
      yield { type: "stderr" as const, data: "err\n" };
      yield { type: "complete" as const, exitCode: 1 };
    }
    mockExecute.mockReturnValue(fakeExecute());

    const { result } = renderHook(() => useExecute("sess1", termRef));

    await act(async () => {
      await result.current.run("bad");
    });

    expect(termRef.current.write).toHaveBeenCalledWith("\x1b[31merr\n\x1b[0m");
    expect(termRef.current.writeln).toHaveBeenCalledWith("\x1b[31mexit code: 1\x1b[0m");
  });

  it("does not run when sessionId is null", async () => {
    const termRef = makeTerminalRef();

    const { result } = renderHook(() => useExecute(null, termRef));

    await act(async () => {
      await result.current.run("cmd");
    });

    expect(mockExecute).not.toHaveBeenCalled();
  });

  it("prevents concurrent executions", async () => {
    const termRef = makeTerminalRef();

    let resolveFirst!: () => void;
    const firstPromise = new Promise<void>((resolve) => {
      resolveFirst = resolve;
    });

    async function* slowExecute() {
      await firstPromise;
      yield { type: "complete" as const, exitCode: 0 };
    }
    mockExecute.mockReturnValue(slowExecute());

    const { result } = renderHook(() => useExecute("sess1", termRef));

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

  it("displays error message in terminal on execute failure", async () => {
    const termRef = makeTerminalRef();

    async function* failingExecute() {
      throw new Error("connection refused");
      yield { type: "complete" as const, exitCode: 0 };
    }
    mockExecute.mockReturnValue(failingExecute());

    const { result } = renderHook(() => useExecute("sess1", termRef));

    await act(async () => {
      await result.current.run("bad-cmd");
    });

    expect(termRef.current.writeln).toHaveBeenCalledWith(
      "\x1b[31mError: connection refused\x1b[0m",
    );
  });
});
