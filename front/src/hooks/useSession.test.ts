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

  it("deletes session on unmount", async () => {
    const { result, unmount } = renderHook(() => useSession());

    await waitFor(() => {
      expect(result.current).toBe("test-session");
    });

    unmount();
    expect(mockDeleteSession).toHaveBeenCalledWith("test-session");
  });
});
