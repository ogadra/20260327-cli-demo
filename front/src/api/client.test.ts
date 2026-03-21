import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createSession, deleteSession, execute } from "./client";

const mockFetch = vi.fn();

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("createSession", () => {
  it("returns sessionId on success", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ sessionId: "abc123" }),
    });

    const result = await createSession();
    expect(result.sessionId).toBe("abc123");
    expect(mockFetch).toHaveBeenCalledWith("/api/session", { method: "POST" });
  });

  it("throws on failure", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 500 });
    await expect(createSession()).rejects.toThrow("Failed to create session: 500");
  });
});

describe("deleteSession", () => {
  it("sends DELETE with session header and keepalive", () => {
    mockFetch.mockResolvedValue({ ok: true });
    deleteSession("abc123");
    expect(mockFetch).toHaveBeenCalledWith("/api/session", {
      method: "DELETE",
      headers: { "X-Session-Id": "abc123" },
      keepalive: true,
    });
  });
});

describe("execute", () => {
  it("yields SSE events from stream", async () => {
    const chunks = [
      'data: {"type":"stdout","data":"hello\\n"}\n\n',
      'data: {"type":"complete","exitCode":0}\n\n',
    ];
    const encoder = new TextEncoder();
    const iterator = chunks[Symbol.iterator]();

    const readable = new ReadableStream({
      pull(controller) {
        const { done, value } = iterator.next();
        if (done) {
          controller.close();
        } else {
          controller.enqueue(encoder.encode(value));
        }
      },
    });

    mockFetch.mockResolvedValue({
      ok: true,
      body: readable,
    });

    const events = [];
    for await (const event of execute("abc123", "echo hello")) {
      events.push(event);
    }

    expect(events).toEqual([
      { type: "stdout", data: "hello\n" },
      { type: "complete", exitCode: 0 },
    ]);
    expect(mockFetch).toHaveBeenCalledWith("/api/execute", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Session-Id": "abc123",
      },
      body: '{"command":"echo hello"}',
    });
  });

  it("throws on HTTP error", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 400 });
    const gen = execute("abc123", "bad");
    await expect(gen.next()).rejects.toThrow("Failed to execute: 400");
  });

  it("throws when body is null", async () => {
    mockFetch.mockResolvedValue({ ok: true, body: null });
    const gen = execute("abc123", "cmd");
    await expect(gen.next()).rejects.toThrow("No response body");
  });
});
