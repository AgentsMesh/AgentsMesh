import { describe, expect, it } from "vitest";

import type { Block } from "@/lib/viewModels/blockstore";

import { pageDisplayMeta } from "./pageDisplayMeta";

function block(data: Record<string, unknown>, text: string | null = null): Block {
  return { data, text } as unknown as Block;
}

describe("pageDisplayMeta", () => {
  it("returns Untitled for a missing block", () => {
    expect(pageDisplayMeta(undefined)).toEqual({ title: "Untitled" });
    expect(pageDisplayMeta(null)).toEqual({ title: "Untitled" });
  });

  it("prefers trimmed data.title", () => {
    expect(pageDisplayMeta(block({ title: "  Hello  " })).title).toBe("Hello");
  });

  it("falls back to trimmed text when title is blank", () => {
    expect(pageDisplayMeta(block({ title: "   " }, "  Body  ")).title).toBe("Body");
  });

  it("falls back to Untitled when both are blank", () => {
    expect(pageDisplayMeta(block({}, "   ")).title).toBe("Untitled");
  });

  it("extracts a string icon and ignores non-string icons", () => {
    expect(pageDisplayMeta(block({ title: "X", icon: "📈" })).icon).toBe("📈");
    expect(pageDisplayMeta(block({ title: "X", icon: 42 })).icon).toBeUndefined();
  });
});
