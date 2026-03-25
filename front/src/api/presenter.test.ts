import { describe, expect, it } from "vitest";
import { MessageType, parsePresenterMessage } from "./presenter";

describe("parsePresenterMessage", () => {
  it("parses slide_sync message", () => {
    expect(parsePresenterMessage(JSON.stringify({ type: MessageType.SlideSync, page: 3 }))).toEqual(
      { type: "slide_sync", page: 3 },
    );
  });

  it("parses hands_on message", () => {
    const msg = { type: MessageType.HandsOn, instruction: "run echo", placeholder: "$ echo hi" };
    expect(parsePresenterMessage(JSON.stringify(msg))).toEqual(msg);
  });

  it("parses viewer_count message", () => {
    expect(
      parsePresenterMessage(JSON.stringify({ type: MessageType.ViewerCount, count: 42 })),
    ).toEqual({ type: "viewer_count", count: 42 });
  });

  it("returns null for unknown type", () => {
    expect(parsePresenterMessage(JSON.stringify({ type: "unknown" }))).toBeNull();
  });

  it("returns null for malformed JSON", () => {
    expect(parsePresenterMessage("not json")).toBeNull();
  });

  it("returns null for missing type field", () => {
    expect(parsePresenterMessage(JSON.stringify({ page: 1 }))).toBeNull();
  });

  it("returns null for non-object values", () => {
    expect(parsePresenterMessage(JSON.stringify("string"))).toBeNull();
    expect(parsePresenterMessage(JSON.stringify(null))).toBeNull();
    expect(parsePresenterMessage(JSON.stringify(42))).toBeNull();
  });

  it("returns null for invalid slide_sync payload shape", () => {
    expect(
      parsePresenterMessage(JSON.stringify({ type: MessageType.SlideSync, page: "3" })),
    ).toBeNull();
  });

  it("returns null for invalid viewer_count payload shape", () => {
    expect(
      parsePresenterMessage(JSON.stringify({ type: MessageType.ViewerCount, count: "42" })),
    ).toBeNull();
  });

  it("returns null for invalid hands_on payload shape", () => {
    expect(
      parsePresenterMessage(
        JSON.stringify({ type: MessageType.HandsOn, instruction: 123, placeholder: null }),
      ),
    ).toBeNull();
  });
});
