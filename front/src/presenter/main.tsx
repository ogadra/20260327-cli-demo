import { StrictMode, useCallback, useEffect, useState, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { usePresenter } from "../hooks/usePresenter";
import { LoginForm } from "./LoginForm";
import { PresenterPanel } from "./PresenterPanel";
import { defaultPolls } from "./sequence";

/** WebSocket URL derived from the current page origin. */
const wsUrl = (): string => location.origin.replace(/^http/, "ws") + "/ws";

/** Authentication state: checking session, logged in, or not logged in. */
type AuthState = "checking" | "loggedIn" | "loggedOut";

/** Root component for the presenter page with login gate. */
const PresenterApp = (): ReactNode => {
  const [authState, setAuthState] = useState<AuthState>("checking");

  /** Check session validity on mount by calling GET /login. */
  useEffect((): void => {
    fetch("/login", { credentials: "include" })
      .then((res): void => {
        setAuthState(res.status === 200 ? "loggedIn" : "loggedOut");
      })
      .catch((): void => {
        setAuthState("loggedOut");
      });
  }, []);

  /** Handle successful login by updating auth state. */
  const handleLoginSuccess = useCallback((): void => {
    setAuthState("loggedIn");
  }, []);

  if (authState === "checking") {
    return null;
  }

  if (authState === "loggedOut") {
    return <LoginForm onSuccess={handleLoginSuccess} />;
  }

  return <PresenterAppInner />;
};

/** Inner component rendered after login that connects the usePresenter hook to the PresenterPanel. */
const PresenterAppInner = (): ReactNode => {
  const { viewerCount, sendSlideSync, sendPollGet } = usePresenter(wsUrl());

  /** Send all poll definitions on mount to initialize polls. */
  useEffect((): void => {
    for (const poll of defaultPolls) {
      sendPollGet(poll.pollId, poll.options, poll.maxChoices);
    }
  }, [sendPollGet]);

  return <PresenterPanel sendSlideSync={sendSlideSync} viewerCount={viewerCount} />;
};

const root = document.getElementById("presenter-root");
if (root) {
  createRoot(root).render(
    <StrictMode>
      <PresenterApp />
    </StrictMode>,
  );
}
