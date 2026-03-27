import type { ReactNode } from "react";
import PollView from "../components/PollView";
import type { SlideProps } from "../components/SlideView";
import { parseInline } from "./parseInline";

/** Props for the PollSlide component. */
interface PollSlideProps extends SlideProps {
  /** Unique identifier for the poll. */
  pollId: string;
  /** Question text displayed above the poll. */
  question: string;
  /** Available answer options. */
  options: string[];
}

/** Slide component that displays a poll question with voting buttons. */
export const PollSlide = ({
  pollId,
  question,
  options,
  pollStates,
  onPollVote,
  onPollUnvote,
  onPollSwitch,
}: PollSlideProps): ReactNode => {
  const state = pollStates[pollId];

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        width: "100%",
        height: "100%",
        padding: "32px",
        boxSizing: "border-box",
      }}
    >
      <div
        style={{
          color: "#fff",
          fontSize: "min(5vw, 40px)",
          fontFamily: "sans-serif",
          textAlign: "center",
          marginBottom: "32px",
        }}
      >
        {parseInline(question)}
      </div>
      <div style={{ width: "100%", maxWidth: "600px" }}>
        <PollView
          options={options}
          maxChoices={1}
          votes={state?.votes ?? Object.fromEntries(options.map((o) => [o, 0]))}
          myChoices={state?.myChoices ?? []}
          onVote={(choice) => onPollVote(pollId, choice)}
          onUnvote={(choice) => onPollUnvote(pollId, choice)}
          onSwitch={(from, to) => onPollSwitch(pollId, from, to)}
        />
      </div>
    </div>
  );
};
