import type { ReactNode } from "react";
import { useCallback, useEffect, useRef } from "react";
import SlideView from "./components/SlideView";
import { usePresenter } from "./hooks/usePresenter";
import { slides } from "./slides/index";

/** WebSocket URL derived from the current page origin. */
const wsUrl = (): string => location.origin.replace(/^http/, "ws") + "/ws";

/** Root application component that renders all slides vertically and scrolls to the active page. */
const App = (): ReactNode => {
  const { page, viewerCount, pollStates, sendPollVote, sendPollUnvote, sendPollSwitch } =
    usePresenter(wsUrl());

  const sectionRefs = useRef<(HTMLDivElement | null)[]>([]);
  const initialScrollDone = useRef(false);

  /** Stores a section ref at the given index. */
  const setSectionRef = useCallback(
    (index: number) =>
      (el: HTMLDivElement | null): void => {
        sectionRefs.current[index] = el;
      },
    [],
  );

  /** Scroll to the active page when it changes. Uses instant scroll for the first navigation. */
  useEffect((): void => {
    const el = sectionRefs.current[page];
    if (!el) return;
    const behavior = initialScrollDone.current ? "smooth" : "instant";
    initialScrollDone.current = true;
    el.scrollIntoView({ behavior, block: "start" });
  }, [page]);

  return (
    <div
      style={{
        height: "100dvh",
        overflowY: "auto",
        scrollSnapType: "y mandatory",
        background: "#000",
      }}
    >
      <div
        style={{
          position: "fixed",
          top: 8,
          right: 8,
          zIndex: 10,
          color: "#aaa",
          fontSize: "12px",
          fontFamily: "sans-serif",
        }}
      >
        {viewerCount} viewers
      </div>

      {slides.map((_, i) => (
        <div
          key={i}
          ref={setSectionRef(i)}
          style={{
            height: "100dvh",
            scrollSnapAlign: "start",
          }}
        >
          <SlideView
            page={i}
            pollStates={pollStates}
            onPollVote={sendPollVote}
            onPollUnvote={sendPollUnvote}
            onPollSwitch={sendPollSwitch}
          />
        </div>
      ))}
    </div>
  );
};

export default App;
