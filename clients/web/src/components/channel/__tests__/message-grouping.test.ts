import { describe, it, expect } from "vitest";
import { computeGroupFlags } from "../message-grouping";
import type { TransformedMessage } from "../types";

function m(id: number, opts: Partial<TransformedMessage> = {}): TransformedMessage {
  return {
    id,
    body: "x",
    messageType: "text",
    createdAt: "2026-01-01T12:00:00Z",
    user: { id: 1, username: "a" },
    ...opts,
  };
}

describe("computeGroupFlags", () => {
  it("first message always starts a group", () => {
    expect(computeGroupFlags([m(1)]).get(1)).toBe(true);
  });

  it("same sender within 5min → merged (not first in group)", () => {
    const f = computeGroupFlags([
      m(1, { createdAt: "2026-01-01T12:00:00Z" }),
      m(2, { createdAt: "2026-01-01T12:02:00Z" }),
    ]);
    expect(f.get(1)).toBe(true);
    expect(f.get(2)).toBe(false);
  });

  it("same sender beyond 5min → new group", () => {
    const f = computeGroupFlags([
      m(1, { createdAt: "2026-01-01T12:00:00Z" }),
      m(2, { createdAt: "2026-01-01T12:06:00Z" }),
    ]);
    expect(f.get(2)).toBe(true);
  });

  it("different sender → new group", () => {
    const f = computeGroupFlags([
      m(1, { user: { id: 1, username: "a" } }),
      m(2, { user: { id: 2, username: "b" }, createdAt: "2026-01-01T12:01:00Z" }),
    ]);
    expect(f.get(2)).toBe(true);
  });

  it("pod sender keyed by podKey, merges consecutive", () => {
    const f = computeGroupFlags([
      m(1, { user: undefined, pod: { podKey: "p1" }, createdAt: "2026-01-01T12:00:00Z" }),
      m(2, { user: undefined, pod: { podKey: "p1" }, createdAt: "2026-01-01T12:01:00Z" }),
      m(3, { user: undefined, pod: { podKey: "p2" }, createdAt: "2026-01-01T12:02:00Z" }),
    ]);
    expect(f.get(2)).toBe(false); // same pod → merged
    expect(f.get(3)).toBe(true); // different pod → new group
  });

  it("system messages never merge (neither side)", () => {
    const f = computeGroupFlags([
      m(1, { createdAt: "2026-01-01T12:00:00Z" }),
      m(2, { messageType: "system", createdAt: "2026-01-01T12:01:00Z" }),
      m(3, { createdAt: "2026-01-01T12:02:00Z" }),
    ]);
    expect(f.get(2)).toBe(true);
    expect(f.get(3)).toBe(true);
  });

  it("tool_call content never merges", () => {
    const f = computeGroupFlags([
      m(1, { createdAt: "2026-01-01T12:00:00Z" }),
      m(2, {
        content: { kind: "tool_call" } as TransformedMessage["content"],
        createdAt: "2026-01-01T12:01:00Z",
      }),
    ]);
    expect(f.get(2)).toBe(true);
  });
});
