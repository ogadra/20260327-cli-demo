import type { ReactNode } from "react";
import slides from "../slides/index";
import type { PollStateData } from "../hooks/usePresenter";

/** Props passed to each slide component. */
export interface SlideProps {
  pollState: PollStateData | null;
  onPollVote: (choice: string) => void;
  onPollUnvote: (choice: string) => void;
  onPollSwitch: (from: string, to: string) => void;
}

/** Props for the SlideView component. */
interface SlideViewProps extends SlideProps {
  page: number;
}

/** Renders the slide component corresponding to the given page number. */
const SlideView = ({
  page,
  pollState,
  onPollVote,
  onPollUnvote,
  onPollSwitch,
}: SlideViewProps): ReactNode => {
  const Slide = slides[page];
  if (!Slide) {
    return (
      <div
        data-testid="slide-fallback"
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          width: "100%",
          height: "100%",
          color: "#888",
          fontSize: "min(4vw, 24px)",
          fontFamily: "sans-serif",
        }}
      >
        Slide not found
      </div>
    );
  }
  return (
    <div data-testid="slide-content" style={{ width: "100%", height: "100%" }}>
      <Slide
        pollState={pollState}
        onPollVote={onPollVote}
        onPollUnvote={onPollUnvote}
        onPollSwitch={onPollSwitch}
      />
    </div>
  );
};

export default SlideView;
