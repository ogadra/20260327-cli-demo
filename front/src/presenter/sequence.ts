import { Action } from "../api/presenter";

/**
 * Discriminated union type representing a single step in the presentation sequence.
 * Each variant carries the data needed by the presenter control panel to drive that step.
 * Type discriminants reference the direction-independent Action constants.
 */
export type PresenterStep =
  | { type: typeof Action.SlideSync; page: number }
  | { type: typeof Action.HandsOn; instruction: string; placeholder: string }
  | {
      type: typeof Action.PollOpen;
      pollId: string;
      options: string[];
      maxChoices: number;
    };

/** Default presentation sequence used by the presenter control panel. */
export const defaultSequence: PresenterStep[] = [{ type: Action.SlideSync, page: 0 }];
