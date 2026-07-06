import { Action, type SlideSyncPayload } from "../api/presenter";
import { slideData } from "../slides/slideData";

/** Poll open payload used only in presenter steps to initialize a poll for viewers. */
export type PollOpenPayload = {
  type: typeof Action.PollOpen;
  pollId: string;
  options: string[];
  maxChoices: number;
};

/**
 * Discriminated union type representing a single display-mode step in the presentation sequence.
 * SlideSyncPayload is shared with server-to-client message types.
 */
export type PresenterStep = SlideSyncPayload | PollOpenPayload;

/** Input shape accepted by buildSequence covering poll slides. */
type BuildSequenceInput = ReadonlyArray<{
  type: string;
  pollId?: string;
  options?: string[];
}>;

/** Generates the presentation sequence from slideData, inserting PollOpen steps after poll slides. */
export const buildSequence = (data: BuildSequenceInput): PresenterStep[] => {
  const steps: PresenterStep[] = [];
  for (let i = 0; i < data.length; i++) {
    steps.push({ type: Action.SlideSync, page: i });
    const slide = data[i];
    if (slide.type === "poll" && slide.pollId && slide.options) {
      steps.push({
        type: Action.PollOpen,
        pollId: slide.pollId,
        options: slide.options,
        maxChoices: 1,
      });
    }
  }
  return steps;
};

/** Default presentation sequence generated from slideData. */
export const defaultSequence: PresenterStep[] = buildSequence(slideData);
