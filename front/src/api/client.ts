export interface SessionResponse {
  sessionId: string;
}

export type SseEvent =
  | { type: "stdout"; data: string }
  | { type: "stderr"; data: string }
  | { type: "complete"; exitCode: number };

export const createSession = async (): Promise<SessionResponse> => {
  const res = await fetch("/api/session", { method: "POST" });
  if (!res.ok) throw new Error(`Failed to create session: ${res.status}`);
  return res.json() as Promise<SessionResponse>;
};

export const deleteSession = (sessionId: string): void => {
  void fetch("/api/session", {
    method: "DELETE",
    headers: { "X-Session-Id": sessionId },
    keepalive: true,
  });
};

// async generator functions cannot be arrow functions
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

  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(decoder.decode(value, { stream: true }));

    const lines = chunks.join("").split("\n");
    chunks.length = 0;
    chunks.push(lines.pop()!);

    for (const line of lines) {
      if (!line.startsWith("data: ")) continue;
      const json = line.slice(6);
      yield JSON.parse(json) as SseEvent;
    }
  }
}
