/** Discriminant values for server-to-client presenter messages. */
export const MessageType = {
  SlideSync: "slide_sync",
  HandsOn: "hands_on",
  ViewerCount: "viewer_count",
  PollState: "poll_state",
  PollError: "poll_error",
} as const;

/** Discriminated union of all server-to-client presenter messages. */
export type PresenterMessage =
  | { type: typeof MessageType.SlideSync; page: number }
  | { type: typeof MessageType.HandsOn; instruction: string; placeholder: string }
  | { type: typeof MessageType.ViewerCount; count: number }
  | {
      type: typeof MessageType.PollState;
      pollId: string;
      options: string[];
      maxChoices: number;
      votes: Record<string, number>;
      myChoices: string[];
    }
  | {
      type: typeof MessageType.PollError;
      pollId: string;
      error: string;
      votes: Record<string, number>;
      myChoices: string[];
    };

/** Display mode driven by the presenter. */
export type PresenterMode = typeof MessageType.SlideSync | typeof MessageType.HandsOn;

/** Check whether a value is an array of strings. */
const isStringArray = (v: unknown): v is string[] =>
  Array.isArray(v) && v.every((item) => typeof item === "string");

/** Check whether a value is a record with string keys and number values. */
const isVotesRecord = (v: unknown): v is Record<string, number> =>
  typeof v === "object" &&
  v !== null &&
  !Array.isArray(v) &&
  Object.values(v as Record<string, unknown>).every((val) => typeof val === "number");

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
    if (msg.type === MessageType.PollState) {
      return typeof msg.pollId === "string" &&
        isStringArray(msg.options) &&
        typeof msg.maxChoices === "number" &&
        isVotesRecord(msg.votes) &&
        isStringArray(msg.myChoices)
        ? {
            type: MessageType.PollState,
            pollId: msg.pollId as string,
            options: msg.options as string[],
            maxChoices: msg.maxChoices as number,
            votes: msg.votes as Record<string, number>,
            myChoices: msg.myChoices as string[],
          }
        : null;
    }
    if (msg.type === MessageType.PollError) {
      return typeof msg.pollId === "string" &&
        typeof msg.error === "string" &&
        isVotesRecord(msg.votes) &&
        isStringArray(msg.myChoices)
        ? {
            type: MessageType.PollError,
            pollId: msg.pollId as string,
            error: msg.error as string,
            votes: msg.votes as Record<string, number>,
            myChoices: msg.myChoices as string[],
          }
        : null;
    }
    return null;
  } catch {
    return null;
  }
};
