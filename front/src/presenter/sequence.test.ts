import { describe, expect, it } from "vitest";
import { Action } from "../api/presenter";
import { buildSequence, defaultPolls, defaultSequence, type PresenterStep } from "./sequence";
import { slideData } from "../slides/slideData";

describe("buildSequence", () => {
  /** Verify that an empty input produces an empty sequence. */
  it("returns empty array for empty input", () => {
    expect(buildSequence([])).toEqual([]);
  });

  /** Verify that non-terminal slides produce only SlideSync steps. */
  it("generates SlideSync for non-terminal slides", () => {
    const data = [{ type: "title" }, { type: "text" }, { type: "poll" }];
    expect(buildSequence(data)).toEqual([
      { type: Action.SlideSync, page: 0 },
      { type: Action.SlideSync, page: 1 },
      { type: Action.SlideSync, page: 2 },
    ]);
  });

  /** Verify that terminal slides produce SlideSync followed by HandsOn. */
  it("inserts HandsOn step after terminal slides", () => {
    const data = [
      { type: "text" },
      { type: "terminal", instruction: "Run it", commands: ["echo hello", "date"] },
      { type: "text" },
    ];
    expect(buildSequence(data)).toEqual([
      { type: Action.SlideSync, page: 0 },
      { type: Action.SlideSync, page: 1 },
      { type: Action.HandsOn, instruction: "Run it", placeholder: "echo hello\ndate" },
      { type: Action.SlideSync, page: 2 },
    ]);
  });

  /** Verify that terminal slides with empty instruction and commands produce correct defaults. */
  it("handles terminal slide with empty instruction and commands", () => {
    const data = [{ type: "terminal", instruction: "", commands: [] }];
    expect(buildSequence(data)).toEqual([
      { type: Action.SlideSync, page: 0 },
      { type: Action.HandsOn, instruction: "", placeholder: "" },
    ]);
  });

  /** Verify that terminal slides without optional fields default gracefully. */
  it("handles terminal slide with missing optional fields", () => {
    const data = [{ type: "terminal" }];
    expect(buildSequence(data)).toEqual([
      { type: Action.SlideSync, page: 0 },
      { type: Action.HandsOn, instruction: "", placeholder: "" },
    ]);
  });
});

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

  /** Verify that the sequence length matches slideData count plus HandsOn steps for terminal slides. */
  it("has correct length based on slideData", () => {
    const terminalCount = slideData.filter((s) => s.type === "terminal").length;
    expect(defaultSequence).toHaveLength(slideData.length + terminalCount);
  });

  /** Verify that every slide page index appears as a SlideSync step. */
  it("contains a SlideSync step for every slide", () => {
    const slideSyncPages = defaultSequence
      .filter(
        (s): s is { type: typeof Action.SlideSync; page: number } => s.type === Action.SlideSync,
      )
      .map((s) => s.page);
    const expectedPages = slideData.map((_, i) => i);
    expect(slideSyncPages).toEqual(expectedPages);
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
