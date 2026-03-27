import { describe, expect, it } from "vitest";
import { Action } from "../api/presenter";
import { defaultPolls, defaultSequence } from "./sequence";

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

  /** Verify that the default sequence contains 3 slide_sync steps. */
  it("contains 3 steps in the expected order", () => {
    expect(defaultSequence).toHaveLength(3);
    expect(defaultSequence).toEqual([
      { type: Action.SlideSync, page: 0 },
      { type: Action.SlideSync, page: 1 },
      { type: Action.SlideSync, page: 2 },
    ]);
  });

  /** Verify that all steps are slide_sync type. */
  it("only contains slide_sync steps", () => {
    for (const step of defaultSequence) {
      expect(step.type).toBe(Action.SlideSync);
    }
  });
});

describe("defaultPolls", () => {
  /** Verify that the default polls list is an empty array. */
  it("is an empty array", () => {
    expect(defaultPolls).toEqual([]);
  });
});
