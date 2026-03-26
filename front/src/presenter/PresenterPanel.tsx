import { type ReactNode, useCallback, useEffect, useRef, useState } from "react";
import { Action } from "../api/presenter";
import type { PollStateData } from "../hooks/usePresenter";
import { defaultSequence, type PresenterStep } from "./sequence";

/** Props for the PresenterPanel component. */
export interface PresenterPanelProps {
  /** Sends a slide_sync message to synchronize viewers to a given page. */
  sendSlideSync: (page: number) => void;
  /** Sends a hands_on message with instruction and placeholder text. */
  sendHandsOn: (instruction: string, placeholder: string) => void;
  /** Sends a poll_get message to initialize or retrieve a poll. */
  sendPollGet: (pollId: string, options: string[], maxChoices: number) => void;
  /** Number of currently connected viewers. */
  viewerCount: number;
  /** Poll states keyed by pollId. */
  pollStates: Partial<Record<string, PollStateData>>;
}

/** Derives a human-readable description from a presenter step. */
const describeStep = (step: PresenterStep): string => {
  switch (step.type) {
    case Action.SlideSync:
      return `Slide ${step.page}`;
    case Action.HandsOn:
      return `Hands-on: ${step.instruction}`;
    case Action.PollOpen:
      return `Poll: ${step.pollId}`;
  }
};

/** Presenter control panel that drives the presentation sequence via step navigation. */
export const PresenterPanel = ({
  sendSlideSync,
  sendHandsOn,
  sendPollGet,
  viewerCount,
  pollStates,
}: PresenterPanelProps): ReactNode => {
  const sequence = defaultSequence;
  const [stepIndex, setStepIndex] = useState(0);
  const lastExecutedRef = useRef<number | null>(null);

  /** Executes the send function corresponding to a given step. */
  const executeStep = useCallback(
    (step: PresenterStep): void => {
      switch (step.type) {
        case Action.SlideSync:
          sendSlideSync(step.page);
          break;
        case Action.HandsOn:
          sendHandsOn(step.instruction, step.placeholder);
          break;
        case Action.PollOpen:
          sendPollGet(step.pollId, step.options, step.maxChoices);
          break;
      }
    },
    [sendSlideSync, sendHandsOn, sendPollGet],
  );

  /** Navigates to a specific step index. */
  const goTo = useCallback(
    (index: number): void => {
      if (index < 0 || index >= sequence.length) return;
      setStepIndex(index);
    },
    [sequence.length],
  );

  /** Executes the current step exactly once per step transition. */
  useEffect((): void => {
    if (lastExecutedRef.current === stepIndex) return;
    lastExecutedRef.current = stepIndex;
    executeStep(sequence[stepIndex]);
  }, [stepIndex, executeStep, sequence]);

  /** Listen for keyboard navigation. */
  useEffect((): (() => void) => {
    /** Handles keydown events for arrow key navigation, ignoring events from form elements. */
    const handleKeyDown = (e: KeyboardEvent): void => {
      const tag = (e.target as HTMLElement).tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
      if (e.key === "ArrowRight") {
        setStepIndex((prev) => {
          const next = prev + 1;
          return next >= sequence.length ? prev : next;
        });
      } else if (e.key === "ArrowLeft") {
        setStepIndex((prev) => {
          const next = prev - 1;
          return next < 0 ? prev : next;
        });
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return (): void => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [sequence.length]);

  const currentStep = sequence[stepIndex];
  const pollState =
    currentStep.type === Action.PollOpen ? pollStates[currentStep.pollId] : undefined;

  return (
    <div
      style={{
        background: "#1a1a1a",
        color: "#fff",
        padding: "24px",
        fontFamily: "sans-serif",
        minHeight: "100vh",
      }}
    >
      <header
        role="status"
        style={{
          display: "flex",
          justifyContent: "space-between",
          marginBottom: "24px",
          fontSize: "14px",
          color: "#aaa",
        }}
      >
        <span>
          Step {stepIndex + 1} / {sequence.length}
        </span>
        <span>{viewerCount} viewers</span>
      </header>

      <div style={{ fontSize: "20px", marginBottom: "24px" }}>{describeStep(currentStep)}</div>

      {pollState && (
        <section aria-label="poll results" style={{ marginBottom: "24px" }}>
          <div style={{ fontSize: "14px", color: "#aaa", marginBottom: "8px" }}>Poll Results</div>
          {pollState.options.map((option) => (
            <div
              key={option}
              style={{
                display: "flex",
                justifyContent: "space-between",
                padding: "8px",
                background: "#222",
                borderRadius: "4px",
                marginBottom: "4px",
              }}
            >
              <span>{option}</span>
              <span style={{ color: "#aaa" }}>{pollState.votes[option] ?? 0}</span>
            </div>
          ))}
        </section>
      )}

      <div style={{ display: "flex", gap: "12px" }}>
        <button
          type="button"
          disabled={stepIndex === 0}
          onClick={(): void => goTo(stepIndex - 1)}
          style={{
            padding: "8px 24px",
            background: stepIndex === 0 ? "#333" : "#555",
            color: stepIndex === 0 ? "#666" : "#fff",
            border: "none",
            borderRadius: "4px",
            cursor: stepIndex === 0 ? "not-allowed" : "pointer",
            fontSize: "16px",
          }}
        >
          Prev
        </button>
        <button
          type="button"
          disabled={stepIndex === sequence.length - 1}
          onClick={(): void => goTo(stepIndex + 1)}
          style={{
            padding: "8px 24px",
            background: stepIndex === sequence.length - 1 ? "#333" : "#555",
            color: stepIndex === sequence.length - 1 ? "#666" : "#fff",
            border: "none",
            borderRadius: "4px",
            cursor: stepIndex === sequence.length - 1 ? "not-allowed" : "pointer",
            fontSize: "16px",
          }}
        >
          Next
        </button>
      </div>
    </div>
  );
};
