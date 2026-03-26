import { describe, expect, it } from "vitest";
import { defaultSequence, type PresenterStep } from "./sequence";

describe("defaultSequence", () => {
  /** Verify that the default sequence is a non-empty array. */
  it("is a non-empty array", () => {
    expect(Array.isArray(defaultSequence)).toBe(true);
    expect(defaultSequence.length).toBeGreaterThan(0);
  });

  /** Verify that the first step is a slide_sync step targeting page 0. */
  it("starts with a slide_sync step at page 0", () => {
    const first = defaultSequence[0];
    expect(first).toEqual({ type: "slide_sync", page: 0 });
  });
});

describe("PresenterStep discriminated union", () => {
  /** Verify that a slide_sync step carries the page property. */
  it("allows slide_sync with page", () => {
    const step: PresenterStep = { type: "slide_sync", page: 3 };
    expect(step.type).toBe("slide_sync");
    if (step.type === "slide_sync") {
      expect(step.page).toBe(3);
    }
  });

  /** Verify that a hands_on step carries instruction and placeholder properties. */
  it("allows hands_on with instruction and placeholder", () => {
    const step: PresenterStep = {
      type: "hands_on",
      instruction: "Run the command",
      placeholder: "echo hello",
    };
    expect(step.type).toBe("hands_on");
    if (step.type === "hands_on") {
      expect(step.instruction).toBe("Run the command");
      expect(step.placeholder).toBe("echo hello");
    }
  });

  /** Verify that a poll_get step carries pollId, options, and maxChoices properties. */
  it("allows poll_get with pollId, options, and maxChoices", () => {
    const step: PresenterStep = {
      type: "poll_get",
      pollId: "poll-1",
      options: ["Yes", "No"],
      maxChoices: 1,
    };
    expect(step.type).toBe("poll_get");
    if (step.type === "poll_get") {
      expect(step.pollId).toBe("poll-1");
      expect(step.options).toEqual(["Yes", "No"]);
      expect(step.maxChoices).toBe(1);
    }
  });

  /** Verify that the type field correctly narrows the union via switch. */
  it("narrows correctly via switch on type", () => {
    const steps: PresenterStep[] = [
      { type: "slide_sync", page: 1 },
      { type: "hands_on", instruction: "do it", placeholder: "cmd" },
      { type: "poll_get", pollId: "p1", options: ["A"], maxChoices: 1 },
    ];

    const types = steps.map((s) => s.type);
    expect(types).toEqual(["slide_sync", "hands_on", "poll_get"]);
  });
});
