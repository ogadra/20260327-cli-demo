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
    expect(mockFetch).toHaveBeenCalledWith("/api/session", {
      method: "POST",
      credentials: "include",
      signal: undefined,
    });
  });

  it("forwards AbortSignal to fetch", async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ sessionId: "abc123" }),
    });
    const signal = new AbortController().signal;
    await createSession(signal);
    expect(mockFetch).toHaveBeenCalledWith("/api/session", {
      method: "POST",
      credentials: "include",
      signal,
    });
  });

  it("throws on failure", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 500 });
    await expect(createSession()).rejects.toThrow("Failed to create session: 500");
  });
});

describe("deleteSession", () => {
  it("sends DELETE with credentials and keepalive", () => {
    mockFetch.mockResolvedValue({ ok: true });
    deleteSession();
    expect(mockFetch).toHaveBeenCalledWith("/api/session", {
      method: "DELETE",
      credentials: "include",
      keepalive: true,
    });
  });

  it("logs error when fetch rejects", async () => {
    const error = new Error("network failure");
    mockFetch.mockRejectedValue(error);
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    deleteSession();
    await new Promise((r) => setTimeout(r, 0));

    expect(spy).toHaveBeenCalledWith("Failed to delete session", error);
    spy.mockRestore();
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
    for await (const event of execute("echo hello")) {
      events.push(event);
    }

    expect(events).toEqual([
      { type: "stdout", data: "hello\n" },
      { type: "complete", exitCode: 0 },
    ]);
    expect(mockFetch).toHaveBeenCalledWith("/api/execute", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: '{"command":"echo hello"}',
    });
  });

  it("yields event from stream chunk without trailing newline", async () => {
    const chunks = [
      'data: {"type":"stdout","data":"hello\\n"}\n\n',
      'data: {"type":"complete","exitCode":0}',
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

    mockFetch.mockResolvedValue({ ok: true, body: readable });

    const events = [];
    for await (const event of execute("echo hello")) {
      events.push(event);
    }

    expect(events).toEqual([
      { type: "stdout", data: "hello\n" },
      { type: "complete", exitCode: 0 },
    ]);
  });

  it("throws on HTTP error", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 400 });
    const gen = execute("bad");
    await expect(gen.next()).rejects.toThrow("Failed to execute: 400");
  });

  it("throws when body is null", async () => {
    mockFetch.mockResolvedValue({ ok: true, body: null });
    const gen = execute("cmd");
    await expect(gen.next()).rejects.toThrow("No response body");
  });

  it("cancels reader when consumer breaks early", async () => {
    const cancelFn = vi.fn();
    let readCount = 0;
    const readable = new ReadableStream({
      pull(controller) {
        readCount++;
        if (readCount === 1) {
          controller.enqueue(
            new TextEncoder().encode('data: {"type":"stdout","data":"line1\\n"}\n\n'),
          );
        }
      },
      cancel: cancelFn,
    });

    mockFetch.mockResolvedValue({ ok: true, body: readable });

    const events = [];
    for await (const event of execute("cmd")) {
      events.push(event);
      break;
    }

    expect(events).toHaveLength(1);
    expect(cancelFn).toHaveBeenCalled();
  });
});
