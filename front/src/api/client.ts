/** Response returned by the session creation endpoint. */
export interface SessionResponse {
  /** Unique identifier for the created session. */
  sessionId: string;
}

/** Discriminant values for server-sent events. */
export const SseEventType = {
  STDOUT: "stdout",
  STDERR: "stderr",
  COMPLETE: "complete",
} as const;

/** Discriminated union of all server-sent event payloads. */
export type SseEvent =
  | { type: typeof SseEventType.STDOUT; data: string }
  | { type: typeof SseEventType.STDERR; data: string }
  | { type: typeof SseEventType.COMPLETE; exitCode: number };

/**
 * Create a new session on the server.
 * @param signal - Optional AbortSignal to cancel the request.
 */
export const createSession = async (signal?: AbortSignal): Promise<SessionResponse> => {
  const res = await fetch("/api/session", { method: "POST", signal });
  if (!res.ok) throw new Error(`Failed to create session: ${res.status}`);
  return res.json() as Promise<SessionResponse>;
};

/**
 * Delete a session. Errors are logged but not thrown since this is
 * typically called during page unload.
 * @param sessionId - The session to delete.
 */
export const deleteSession = (sessionId: string): void => {
  void fetch("/api/session", {
    method: "DELETE",
    headers: { "X-Session-Id": sessionId },
    keepalive: true,
  }).catch((err: unknown) => {
    console.error("Failed to delete session", err);
  });
};

/**
 * Execute a command in the given session and yield SSE events as they arrive.
 * The reader is automatically cancelled if the consumer exits early.
 * @param sessionId - The session in which to run the command.
 * @param command - The shell command to execute.
 */
export async function* execute(sessionId: string, command: string): AsyncGenerator<SseEvent> {
  const res = await fetch("/api/execute", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Session-Id": sessionId,
    },
    body: JSON.stringify({ command }),
  });
  if (!res.ok) throw new Error(`Failed to execute: ${res.status}`);
  if (!res.body) throw new Error("No response body");

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  const chunks: string[] = [];
  let completed = false;

  try {
    for (;;) {
      const { done, value } = await reader.read();
      chunks.push(done ? decoder.decode() : decoder.decode(value, { stream: true }));

      const lines = chunks.join("").split("\n");
      chunks.length = 0;
      if (!done) chunks.push(lines.pop()!);

      for (const line of lines) {
        if (!line.startsWith("data: ")) continue;
        yield JSON.parse(line.slice(6)) as SseEvent;
      }

      if (done) {
        completed = true;
        break;
      }
    }
  } finally {
    if (!completed) {
      await reader.cancel();
    }
  }
}
