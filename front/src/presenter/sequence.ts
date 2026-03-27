import { Action, type HandsOnPayload, type SlideSyncPayload } from "../api/presenter";
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
 * SlideSyncPayload and HandsOnPayload are shared with server-to-client message types.
 */
export type PresenterStep = SlideSyncPayload | HandsOnPayload | PollOpenPayload;

/** Generates the presentation sequence from slideData, inserting HandsOn steps after terminal slides. */
export const buildSequence = (
  data: ReadonlyArray<{ type: string; instruction?: string; commands?: string[] }>,
): PresenterStep[] => {
  const steps: PresenterStep[] = [];
  for (let i = 0; i < data.length; i++) {
    steps.push({ type: Action.SlideSync, page: i });
    const slide = data[i];
    if (slide.type === "terminal") {
      steps.push({
        type: Action.HandsOn,
        instruction: slide.instruction ?? "",
        placeholder: (slide.commands ?? []).join("\n"),
      });
    }
  }
  return steps;
};

/** Default presentation sequence generated from slideData. */
export const defaultSequence: PresenterStep[] = buildSequence(slideData);

/** Default poll list reserved for future use. */
export const defaultPolls: PollOpenPayload[] = [];
