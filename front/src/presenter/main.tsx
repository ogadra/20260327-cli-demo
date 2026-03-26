import { StrictMode, useEffect, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { Action } from "../api/presenter";
import { usePresenter } from "../hooks/usePresenter";
import { PresenterPanel } from "./PresenterPanel";
import { defaultSequence } from "./sequence";

/** WebSocket URL derived from the current page origin. */
const wsUrl = (): string => location.origin.replace(/^http/, "ws") + "/ws";

/** Root component for the presenter page that connects the usePresenter hook to the PresenterPanel. */
const PresenterApp = (): ReactNode => {
  const { viewerCount, pollStates, sendSlideSync, sendHandsOn, sendPollGet } =
    usePresenter(wsUrl());

  /** Send all poll_open steps on mount to initialize polls. */
  useEffect((): void => {
    for (const step of defaultSequence) {
      if (step.type === Action.PollOpen) {
        sendPollGet(step.pollId, step.options, step.maxChoices);
      }
    }
  }, [sendPollGet]);

  return (
    <PresenterPanel
      sendSlideSync={sendSlideSync}
      sendHandsOn={sendHandsOn}
      sendPollGet={sendPollGet}
      viewerCount={viewerCount}
      pollStates={pollStates}
    />
  );
};

const root = document.getElementById("presenter-root");
if (root) {
  createRoot(root).render(
    <StrictMode>
      <PresenterApp />
    </StrictMode>,
  );
}
