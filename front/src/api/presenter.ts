/** Discriminant values for server-to-client presenter messages. */
export const MessageType = {
  SlideSync: "slide_sync",
  HandsOn: "hands_on",
  ViewerCount: "viewer_count",
} as const;

/** Discriminated union of all server-to-client presenter messages. */
export type PresenterMessage =
  | { type: typeof MessageType.SlideSync; page: number }
  | { type: typeof MessageType.HandsOn; instruction: string; placeholder: string }
  | { type: typeof MessageType.ViewerCount; count: number };

/** Display mode driven by the presenter. */
export type PresenterMode = typeof MessageType.SlideSync | typeof MessageType.HandsOn;

/**
 * Parse a raw JSON string into a known presenter message.
 * Returns null for malformed, unknown, or non-object payloads.
 * @param raw - Raw JSON string from WebSocket.
 * @returns Parsed message or null.
 */
export const parsePresenterMessage = (raw: string): PresenterMessage | null => {
  try {
    const data: unknown = JSON.parse(raw);
    if (typeof data !== "object" || data === null || !("type" in data)) return null;
    const msg = data as Record<string, unknown>;

    if (msg.type === MessageType.SlideSync) {
      return typeof msg.page === "number" ? { type: MessageType.SlideSync, page: msg.page } : null;
    }
    if (msg.type === MessageType.HandsOn) {
      return typeof msg.instruction === "string" && typeof msg.placeholder === "string"
        ? {
            type: MessageType.HandsOn,
            instruction: msg.instruction,
            placeholder: msg.placeholder,
          }
        : null;
    }
    if (msg.type === MessageType.ViewerCount) {
      return typeof msg.count === "number"
        ? { type: MessageType.ViewerCount, count: msg.count }
        : null;
    }
    return null;
  } catch {
    return null;
  }
};
