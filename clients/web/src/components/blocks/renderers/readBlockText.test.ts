import { describe, expect, it } from "vitest";

import type { Block } from "@/lib/api/blockstoreTypes";

import { readBlockText } from "./readBlockText";

function block(over: Partial<Block>): Block {
  return {
    id: "b", workspace_id: "w", type: "paragraph", data: {},
    meta: {}, created_by: 1, created_at: "", updated_at: "",
    ...over,
  } as Block;
}

describe("readBlockText", () => {
  it("returns data.text when present (canonical UI source)", () => {
    expect(readBlockText(block({ data: { text: "hello" }, text: "summary" }))).toBe("hello");
  });

  it("falls back to top-level text when data.text missing (issue #366)", () => {
    expect(readBlockText(block({ data: {}, text: "from agent" }))).toBe("from agent");
  });

  it("falls back to top-level text when data.text is non-string", () => {
    expect(readBlockText(block({ data: { text: 42 as unknown as string }, text: "summary" }))).toBe("summary");
  });

  it("returns empty string when both missing", () => {
    expect(readBlockText(block({ data: {}, text: null }))).toBe("");
  });

  it("returns empty data.text rather than falling back (writer cleared the field)", () => {
    expect(readBlockText(block({ data: { text: "" }, text: "stale summary" }))).toBe("");
  });
});
