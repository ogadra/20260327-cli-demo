import { describe, expect, it } from "vitest";
import { ClientMessageType, MessageType } from "../api/presenter";
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
    const step: PresenterStep = { type: MessageType.SlideSync, page: 3 };
    expect(step.type).toBe(MessageType.SlideSync);
    if (step.type === MessageType.SlideSync) {
      expect(step.page).toBe(3);
    }
  });

  /** Verify that a hands_on step carries instruction and placeholder properties. */
  it("allows hands_on with instruction and placeholder", () => {
    const step: PresenterStep = {
      type: MessageType.HandsOn,
      instruction: "Run the command",
      placeholder: "echo hello",
    };
    expect(step.type).toBe(MessageType.HandsOn);
    if (step.type === MessageType.HandsOn) {
      expect(step.instruction).toBe("Run the command");
      expect(step.placeholder).toBe("echo hello");
    }
  });

  /** Verify that a poll_get step carries pollId, options, and maxChoices properties. */
  it("allows poll_get with pollId, options, and maxChoices", () => {
    const step: PresenterStep = {
      type: ClientMessageType.PollGet,
      pollId: "poll-1",
      options: ["Yes", "No"],
      maxChoices: 1,
    };
    expect(step.type).toBe(ClientMessageType.PollGet);
    if (step.type === ClientMessageType.PollGet) {
      expect(step.pollId).toBe("poll-1");
      expect(step.options).toEqual(["Yes", "No"]);
      expect(step.maxChoices).toBe(1);
    }
  });

  /** Verify that the type field correctly narrows the union via switch. */
  it("narrows correctly via switch on type", () => {
    const steps: PresenterStep[] = [
      { type: MessageType.SlideSync, page: 1 },
      { type: MessageType.HandsOn, instruction: "do it", placeholder: "cmd" },
      { type: ClientMessageType.PollGet, pollId: "p1", options: ["A"], maxChoices: 1 },
    ];

    const types = steps.map((s) => s.type);
    expect(types).toEqual([MessageType.SlideSync, MessageType.HandsOn, ClientMessageType.PollGet]);
  });
});
