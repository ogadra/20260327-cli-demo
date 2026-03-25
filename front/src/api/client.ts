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
 * The session ID is stored as a cookie by the browser automatically.
 * @param signal - Optional AbortSignal to cancel the request.
 */
export const createSession = async (signal?: AbortSignal): Promise<void> => {
  const res = await fetch("/api/session", { method: "POST", credentials: "include", signal });
  if (!res.ok) throw new Error(`Failed to create session: ${res.status}`);
};

/**
 * Delete the current session identified by the session_id cookie.
 * Errors are logged but not thrown since this is typically called during page unload.
 */
export const deleteSession = (): void => {
  void fetch("/api/session", {
    method: "DELETE",
    credentials: "include",
    keepalive: true,
  }).catch((err: unknown) => {
    console.error("Failed to delete session", err);
  });
};

/**
 * Execute a command in the current session and yield SSE events as they arrive.
 * The session is identified by the session_id cookie sent automatically by the browser.
 * The reader is automatically cancelled if the consumer exits early.
 * @param command - The shell command to execute.
 * @param onReassigned - Optional callback invoked when the session was reassigned due to a dead runner.
 */
export async function* execute(
  command: string,
  onReassigned?: () => void,
): AsyncGenerator<SseEvent> {
  const res = await fetch("/api/execute", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ command }),
  });
  if (!res.ok) throw new Error(`Failed to execute: ${res.status}`);
  if (!res.body) throw new Error("No response body");

  if (res.headers.get("X-Session-Reassigned") === "true" && onReassigned) {
    onReassigned();
  }

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
