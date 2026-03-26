/**
 * Discriminated union type representing a single step in the presentation sequence.
 * Each variant carries the data needed by the presenter control panel to drive that step.
 */
export type PresenterStep =
  | { type: "slide_sync"; page: number }
  | { type: "hands_on"; instruction: string; placeholder: string }
  | { type: "poll_get"; pollId: string; options: string[]; maxChoices: number };

/** Default presentation sequence used by the presenter control panel. */
export const defaultSequence: PresenterStep[] = [{ type: "slide_sync", page: 0 }];
