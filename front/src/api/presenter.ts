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

/** Check whether a value is a string. */
const isString = (v: unknown): v is string => typeof v === "string";

/** Check whether a value is a number. */
const isNumber = (v: unknown): v is number => typeof v === "number";

/** Check whether a value is an array of strings. */
const isStringArray = (v: unknown): v is string[] => Array.isArray(v) && v.every(isString);

/** Check whether a value is a record with string keys and number values. */
const isVotesRecord = (v: unknown): v is Record<string, number> =>
  typeof v === "object" &&
  v !== null &&
  !Array.isArray(v) &&
  Object.values(v as Record<string, unknown>).every(isNumber);

/** Validate that msg has the shape of a SlideSync payload. */
const isSlideSyncPayload = (msg: Record<string, unknown>): boolean => isNumber(msg.page);

/** Validate that msg has the shape of a HandsOn payload. */
const isHandsOnPayload = (msg: Record<string, unknown>): boolean =>
  isString(msg.instruction) && isString(msg.placeholder);

/** Validate that msg has the shape of a ViewerCount payload. */
const isViewerCountPayload = (msg: Record<string, unknown>): boolean => isNumber(msg.count);

/** Validate that msg has the shape of a PollState payload. */
const isPollStatePayload = (msg: Record<string, unknown>): boolean =>
  isString(msg.pollId) &&
  isStringArray(msg.options) &&
  isNumber(msg.maxChoices) &&
  isVotesRecord(msg.votes) &&
  isStringArray(msg.myChoices);

/** Validate that msg has the shape of a PollError payload. */
const isPollErrorPayload = (msg: Record<string, unknown>): boolean =>
  isString(msg.pollId) &&
  isString(msg.error) &&
  isVotesRecord(msg.votes) &&
  isStringArray(msg.myChoices);

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
      return isSlideSyncPayload(msg)
        ? { type: MessageType.SlideSync, page: msg.page as number }
        : null;
    }
    if (msg.type === MessageType.HandsOn) {
      return isHandsOnPayload(msg)
        ? {
            type: MessageType.HandsOn,
            instruction: msg.instruction as string,
            placeholder: msg.placeholder as string,
          }
        : null;
    }
    if (msg.type === MessageType.ViewerCount) {
      return isViewerCountPayload(msg)
        ? { type: MessageType.ViewerCount, count: msg.count as number }
        : null;
    }
    if (msg.type === MessageType.PollState) {
      return isPollStatePayload(msg)
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
      return isPollErrorPayload(msg)
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
