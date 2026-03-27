import { Action, type SlideSyncPayload } from "../api/presenter";

/** Poll open payload used only in presenter steps to initialize a poll for viewers. */
export type PollOpenPayload = {
  type: typeof Action.PollOpen;
  pollId: string;
  options: string[];
  maxChoices: number;
};

/** A single step in the presentation sequence. All steps are slide_sync. */
export type PresenterStep = SlideSyncPayload;

/** Default presentation sequence used by the presenter control panel. */
export const defaultSequence: PresenterStep[] = [
  { type: Action.SlideSync, page: 0 },
  { type: Action.SlideSync, page: 1 },
  { type: Action.SlideSync, page: 2 },
];

/** Default poll list reserved for future use. */
export const defaultPolls: PollOpenPayload[] = [];
