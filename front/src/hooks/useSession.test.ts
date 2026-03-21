import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useSession } from "./useSession";

vi.mock("../api/client", () => ({
  createSession: vi.fn(),
  deleteSession: vi.fn(),
}));

import { createSession, deleteSession } from "../api/client";

const mockCreateSession = vi.mocked(createSession);
const mockDeleteSession = vi.mocked(deleteSession);

beforeEach(() => {
  mockCreateSession.mockReset();
  mockDeleteSession.mockReset();
  mockCreateSession.mockResolvedValue({ sessionId: "test-session" });
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useSession", () => {
  it("creates session on mount and returns sessionId", async () => {
    const { result } = renderHook(() => useSession());
    expect(result.current).toBeNull();

    await waitFor(() => {
      expect(result.current).toBe("test-session");
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

  it("does not set sessionId when createSession fails", async () => {
    mockCreateSession.mockRejectedValue(new Error("network error"));

    const { result } = renderHook(() => useSession());

    await waitFor(() => {
      expect(mockCreateSession).toHaveBeenCalledOnce();
    });
    expect(result.current).toBeNull();
  });

  it("logs non-abort errors to console.error", async () => {
    const error = new Error("network error");
    mockCreateSession.mockRejectedValue(error);
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    renderHook(() => useSession());

    await waitFor(() => {
      expect(spy).toHaveBeenCalledWith("Failed to create session", error);
    });

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

    const { unmount } = renderHook(() => useSession());

    await waitFor(() => {
      expect(mockCreateSession).toHaveBeenCalledOnce();
    });
    unmount();

    expect(mockDeleteSession).not.toHaveBeenCalled();
  });

  it("deletes session on unmount", async () => {
    const { result, unmount } = renderHook(() => useSession());

    await waitFor(() => {
      expect(result.current).toBe("test-session");
    });

    unmount();
    expect(mockDeleteSession).toHaveBeenCalledWith();
  });
});
