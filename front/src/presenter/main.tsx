import { StrictMode, useCallback, useEffect, useState, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { usePresenter } from "../hooks/usePresenter";
import PresenterPanel from "./PresenterPanel";
import LoginForm from "./LoginForm";
import { defaultSequence } from "./sequence";

/** WebSocket URL derived from the current page origin. */
const wsUrl = (): string => location.origin.replace(/^http/, "ws") + "/ws";

/** Root component for the presenter page. */
const PresenterApp = (): ReactNode => {
  const [loggedIn, setLoggedIn] = useState(
    () => sessionStorage.getItem("presenterLoggedIn") === "1",
  );

  /** Handle successful login by storing state in sessionStorage. */
  const handleLoginSuccess = useCallback((): void => {
    sessionStorage.setItem("presenterLoggedIn", "1");
    setLoggedIn(true);
  }, []);

  if (!loggedIn) {
    return <LoginForm onSuccess={handleLoginSuccess} />;
  }

  return <PresenterAppInner />;
};

/** Inner component rendered after login that uses the presenter hook. */
const PresenterAppInner = (): ReactNode => {
  const { viewerCount, pollStates, sendSlideSync, sendHandsOn, sendPollGet } =
    usePresenter(wsUrl());

  /** Send all poll_get steps on mount to initialize polls. */
  useEffect((): void => {
    for (const step of defaultSequence) {
      if (step.type === "poll_get") {
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
