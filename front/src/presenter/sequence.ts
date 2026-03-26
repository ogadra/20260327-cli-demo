import { ClientMessageType, MessageType } from "../api/presenter";

/**
 * Discriminated union type representing a single step in the presentation sequence.
 * Each variant carries the data needed by the presenter control panel to drive that step.
 * Type discriminants are shared with the presenter message types defined in api/presenter.
 */
export type PresenterStep =
  | { type: typeof MessageType.SlideSync; page: number }
  | { type: typeof MessageType.HandsOn; instruction: string; placeholder: string }
  | {
      type: typeof ClientMessageType.PollGet;
      pollId: string;
      options: string[];
      maxChoices: number;
    };

/** Default presentation sequence used by the presenter control panel. */
export const defaultSequence: PresenterStep[] = [{ type: MessageType.SlideSync, page: 0 }];
