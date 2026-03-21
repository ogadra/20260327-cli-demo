import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useExecute } from "./useExecute";

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

function makeTerminalRef() {
  return {
    current: {
      write: vi.fn(),
      writeln: vi.fn(),
    },
  };
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
});
