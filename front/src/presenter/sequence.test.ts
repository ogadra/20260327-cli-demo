import { describe, expect, it } from "vitest";
import { Action } from "../api/presenter";
import { defaultPolls, defaultSequence, type PresenterStep } from "./sequence";

describe("defaultSequence", () => {
  /** Verify that the default sequence is a non-empty array. */
  it("is a non-empty array", () => {
    expect(Array.isArray(defaultSequence)).toBe(true);
    expect(defaultSequence.length).toBeGreaterThan(0);
  });

  /** Verify that the first step is a slide_sync step targeting page 0. */
  it("starts with a slide_sync step at page 0", () => {
    const first = defaultSequence[0];
    expect(first).toEqual({ type: Action.SlideSync, page: 0 });
  });

  /** Verify that the default sequence contains 4 steps in the expected order. */
  it("contains 4 steps in the expected order", () => {
    expect(defaultSequence).toHaveLength(4);
    expect(defaultSequence).toEqual([
      { type: Action.SlideSync, page: 0 },
      { type: Action.SlideSync, page: 1 },
      { type: Action.HandsOn, instruction: "Try running a command", placeholder: "echo hello" },
      { type: Action.SlideSync, page: 2 },
    ]);
  });
});

describe("PresenterStep discriminated union", () => {
  /** Verify that a slide_sync step carries the page property. */
  it("allows slide_sync with page", () => {
    const step: PresenterStep = { type: Action.SlideSync, page: 3 };
    expect(step.type).toBe(Action.SlideSync);
    if (step.type === Action.SlideSync) {
      expect(step.page).toBe(3);
    }
  });

  /** Verify that a hands_on step carries instruction and placeholder properties. */
  it("allows hands_on with instruction and placeholder", () => {
    const step: PresenterStep = {
      type: Action.HandsOn,
      instruction: "Run the command",
      placeholder: "echo hello",
    };
    expect(step.type).toBe(Action.HandsOn);
    if (step.type === Action.HandsOn) {
      expect(step.instruction).toBe("Run the command");
      expect(step.placeholder).toBe("echo hello");
    }
  });

  /** Verify that all step types are present in the expected order. */
  it("collects expected ordered types", () => {
    const steps: PresenterStep[] = [
      { type: Action.SlideSync, page: 1 },
      { type: Action.HandsOn, instruction: "do it", placeholder: "cmd" },
    ];

    const types = steps.map((s) => s.type);
    expect(types).toEqual([Action.SlideSync, Action.HandsOn]);
  });
});

describe("defaultPolls", () => {
  /** Verify that the default polls list is an empty array. */
  it("is an empty array", () => {
    expect(defaultPolls).toEqual([]);
  });
});
