/** Direction-independent action identifiers shared across presenter protocol and presenter-step modeling. */
export const Action = {
  SlideSync: "slide_sync",
  HandsOn: "hands_on",
  ViewerCount: "viewer_count",
  PollState: "poll_state",
  PollError: "poll_error",
  PollGet: "poll_get",
  PollVote: "poll_vote",
  PollUnvote: "poll_unvote",
  PollSwitch: "poll_switch",
  PollOpen: "poll_open",
} as const;

/** Discriminant values for server-to-client presenter messages. */
export const MessageType = {
  SlideSync: Action.SlideSync,
  HandsOn: Action.HandsOn,
  ViewerCount: Action.ViewerCount,
  PollState: Action.PollState,
  PollError: Action.PollError,
} as const;

/** Discriminant values for client-to-server presenter messages. */
export const ClientMessageType = {
  PollGet: Action.PollGet,
  PollVote: Action.PollVote,
  PollUnvote: Action.PollUnvote,
  PollSwitch: Action.PollSwitch,
} as const;

/** Slide page synchronization payload shared by server messages and presenter steps. */
export type SlideSyncPayload = { type: typeof Action.SlideSync; page: number };

/** Hands-on mode payload shared by server messages and presenter steps. */
export type HandsOnPayload = {
  type: typeof Action.HandsOn;
  instruction: string;
  placeholder: string;
};

/** Viewer count notification payload. */
export type ViewerCountPayload = { type: typeof Action.ViewerCount; count: number };

/** Poll state payload. */
export type PollStatePayload = {
  type: typeof Action.PollState;
  pollId: string;
  options: string[];
  maxChoices: number;
  votes: Record<string, number>;
  myChoices: string[];
};

/** Poll error payload. */
export type PollErrorPayload = {
  type: typeof Action.PollError;
  pollId: string;
  error: string;
  votes: Record<string, number>;
  myChoices: string[];
};

/** Discriminated union of all server-to-client presenter messages. */
export type PresenterMessage =
  | SlideSyncPayload
  | HandsOnPayload
  | ViewerCountPayload
  | PollStatePayload
  | PollErrorPayload;

/** Display mode driven by the presenter. */
export type PresenterMode = typeof Action.SlideSync | typeof Action.HandsOn;

/** Check whether a value is a string. */
const isString = (v: unknown): v is string => typeof v === "string";

/** Check whether a value is a number. */
const isNumber = (v: unknown): v is number => typeof v === "number";

/** Check whether a value is a safe integer. */
const isInteger = (v: unknown): v is number => Number.isSafeInteger(v);

/** Check whether a value is an array of strings. */
const isStringArray = (v: unknown): v is string[] => Array.isArray(v) && v.every(isString);

/** Check whether a value is a record with string keys and integer values. */
const isVotesRecord = (v: unknown): v is Record<string, number> =>
  typeof v === "object" &&
  v !== null &&
  !Array.isArray(v) &&
  Object.values(v as Record<string, unknown>).every(isInteger);

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
  isInteger(msg.maxChoices) &&
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
