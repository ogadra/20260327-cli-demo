import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useSession } from "./useSession";

vi.mock("../api/client", () => ({
  createSession: vi.fn(),
  deleteSession: vi.fn(),
}));

import { createSession, deleteSession } from "../api/client";

const mockCreateSession = vi.mocked(createSession);
const mockDeleteSession = vi.mocked(deleteSession);

/** Flush all pending microtasks so async effects settle. */
const flushMicrotasks = async (): Promise<void> => {
  await act(async () => {
    await Promise.resolve();
  });
};

beforeEach(() => {
  mockCreateSession.mockReset();
  mockDeleteSession.mockReset();
  mockCreateSession.mockResolvedValue(undefined);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useSession", () => {
  it("starts with loading status and transitions to ready on success", async () => {
    const { result } = renderHook(() => useSession());
    expect(result.current).toBe("loading");

    await waitFor(() => {
      expect(result.current).toBe("ready");
    });
    expect(mockCreateSession).toHaveBeenCalledOnce();
  });

  it("passes AbortSignal to createSession", async () => {
    renderHook(() => useSession());

    await waitFor(() => {
      expect(mockCreateSession).toHaveBeenCalled();
    });

    expect(mockCreateSession.mock.lastCall?.[0]).toBeInstanceOf(AbortSignal);
  });

  it("transitions to retrying when createSession fails", async () => {
    mockCreateSession.mockRejectedValue(new Error("network error"));
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    const { result, unmount } = renderHook(() => useSession());

    await waitFor(() => {
      expect(result.current).toBe("retrying");
    });

    unmount();
    spy.mockRestore();
  });

  it("retries after delay on failure", async () => {
    vi.useFakeTimers();
    mockCreateSession.mockRejectedValueOnce(new Error("network error"));
    mockCreateSession.mockResolvedValueOnce(undefined);
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    const { result } = renderHook(() => useSession());

    await flushMicrotasks();
    expect(result.current).toBe("retrying");
    expect(mockCreateSession).toHaveBeenCalledOnce();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });

    expect(result.current).toBe("ready");
    expect(mockCreateSession).toHaveBeenCalledTimes(2);

    spy.mockRestore();
    vi.useRealTimers();
  });

  it("uses exponential backoff with cap at 8 seconds", async () => {
    vi.useFakeTimers();
    mockCreateSession.mockRejectedValue(new Error("network error"));
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    renderHook(() => useSession());

    await flushMicrotasks();
    expect(mockCreateSession).toHaveBeenCalledOnce();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000);
    });
    expect(mockCreateSession).toHaveBeenCalledTimes(2);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });
    expect(mockCreateSession).toHaveBeenCalledTimes(3);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(4000);
    });
    expect(mockCreateSession).toHaveBeenCalledTimes(4);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(8000);
    });
    expect(mockCreateSession).toHaveBeenCalledTimes(5);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(8000);
    });
    expect(mockCreateSession).toHaveBeenCalledTimes(6);

    spy.mockRestore();
    vi.useRealTimers();
  });

  it("logs non-abort errors to console.error", async () => {
    const error = new Error("network error");
    mockCreateSession.mockRejectedValue(error);
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    const { unmount } = renderHook(() => useSession());

    await waitFor(() => {
      expect(spy).toHaveBeenCalledWith("Failed to create session", error);
    });

    unmount();
    spy.mockRestore();
  });

  it("does not log AbortError", async () => {
    const abortError = new DOMException("The operation was aborted", "AbortError");
    mockCreateSession.mockRejectedValue(abortError);
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    const { unmount } = renderHook(() => useSession());

    await waitFor(() => {
      expect(mockCreateSession).toHaveBeenCalledOnce();
    });
    unmount();

    expect(spy).not.toHaveBeenCalled();
    spy.mockRestore();
  });

  it("does not call deleteSession when createSession fails and unmounts", async () => {
    mockCreateSession.mockRejectedValue(new Error("network error"));
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    const { unmount } = renderHook(() => useSession());

    await waitFor(() => {
      expect(mockCreateSession).toHaveBeenCalledOnce();
    });
    unmount();

    expect(mockDeleteSession).not.toHaveBeenCalled();
    spy.mockRestore();
  });

  it("deletes session on unmount when ready", async () => {
    const { result, unmount } = renderHook(() => useSession());

    await waitFor(() => {
      expect(result.current).toBe("ready");
    });

    unmount();
    expect(mockDeleteSession).toHaveBeenCalledWith();
  });

  it("cancels pending retry on unmount", async () => {
    vi.useFakeTimers();
    mockCreateSession.mockRejectedValueOnce(new Error("network error"));
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    const { unmount } = renderHook(() => useSession());

    await flushMicrotasks();
    expect(mockCreateSession).toHaveBeenCalledOnce();

    unmount();

    mockCreateSession.mockResolvedValueOnce(undefined);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(mockCreateSession).toHaveBeenCalledOnce();
    spy.mockRestore();
    vi.useRealTimers();
  });
});
