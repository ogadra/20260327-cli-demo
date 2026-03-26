import { Action, type HandsOnPayload, type SlideSyncPayload } from "../api/presenter";

/** Poll open payload used only in presenter steps to initialize a poll for viewers. */
export type PollOpenPayload = {
  type: typeof Action.PollOpen;
  pollId: string;
  options: string[];
  maxChoices: number;
};

/**
 * Discriminated union type representing a single step in the presentation sequence.
 * SlideSyncPayload and HandsOnPayload are shared with server-to-client message types.
 * PollOpenPayload is specific to presenter steps.
 */
export type PresenterStep = SlideSyncPayload | HandsOnPayload | PollOpenPayload;

/** Default presentation sequence used by the presenter control panel. */
export const defaultSequence: PresenterStep[] = [{ type: Action.SlideSync, page: 0 }];
