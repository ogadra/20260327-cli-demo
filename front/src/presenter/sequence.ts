import { Action, type HandsOnPayload, type SlideSyncPayload } from "../api/presenter";

/** Poll open payload used only in presenter steps to initialize a poll for viewers. */
export type PollOpenPayload = {
  type: typeof Action.PollOpen;
  pollId: string;
  options: string[];
  maxChoices: number;
};

/**
 * Discriminated union type representing a single display-mode step in the presentation sequence.
 * SlideSyncPayload and HandsOnPayload are shared with server-to-client message types.
 */
export type PresenterStep = SlideSyncPayload | HandsOnPayload;

/** Default presentation sequence used by the presenter control panel. */
export const defaultSequence: PresenterStep[] = [
  { type: Action.SlideSync, page: 0 },
  { type: Action.SlideSync, page: 1 },
  { type: Action.HandsOn, instruction: "Try running a command", placeholder: "echo hello" },
  { type: Action.SlideSync, page: 2 },
];

/** Default poll list reserved for future use. */
export const defaultPolls: PollOpenPayload[] = [];
