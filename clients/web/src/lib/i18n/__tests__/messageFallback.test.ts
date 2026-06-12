import { describe, it, expect } from "vitest";
import { deepMergeMessages } from "@/lib/i18n/messageFallback";

describe("deepMergeMessages", () => {
  it("locale value wins over en base", () => {
    expect(deepMergeMessages({ a: "en" }, { a: "zh" })).toEqual({ a: "zh" });
  });

  it("en fills keys the locale omits", () => {
    expect(deepMergeMessages({ a: "en", b: "en" }, { a: "zh" })).toEqual({ a: "zh", b: "en" });
  });

  it("merges nested objects recursively (locale wins per-leaf, en fills gaps)", () => {
    const en = { acp: { modeSelector: { bypass: { label: "Bypass" }, ask_dangerous: { label: "Ask Risky" } } } };
    const zh = { acp: { modeSelector: { bypass: { label: "全自动" } } } };
    expect(deepMergeMessages(en, zh)).toEqual({
      acp: { modeSelector: { bypass: { label: "全自动" }, ask_dangerous: { label: "Ask Risky" } } },
    });
  });

  it("replaces arrays, does not merge them", () => {
    expect(deepMergeMessages({ a: [1, 2] }, { a: [3] })).toEqual({ a: [3] });
  });
});
