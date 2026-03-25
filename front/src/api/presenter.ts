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
    const msg = data as PresenterMessage;
    if (
      msg.type === MessageType.SlideSync ||
      msg.type === MessageType.HandsOn ||
      msg.type === MessageType.ViewerCount
    ) {
      return msg;
    }
    return null;
  } catch {
    return null;
  }
};
