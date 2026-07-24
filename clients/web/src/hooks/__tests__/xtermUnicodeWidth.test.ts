import { describe, expect, it } from "vitest";
import { Terminal } from "@xterm/xterm";
import {
  RUNNER_UNICODE_VERSION,
  runnerRuneWidth,
  RunnerUnicodeGraphemesAddon,
} from "@/lib/terminal/runnerUnicodeGraphemes";
import {
  DOUBLE_WIDTH_RANGES,
  ZERO_WIDTH_RANGES,
} from "@/lib/terminal/runewidthUnicode17Ranges";

async function cursorAfter(chunks: string[]): Promise<number> {
  const terminal = new Terminal({ allowProposedApi: true, cols: 40 });
  terminal.loadAddon(new RunnerUnicodeGraphemesAddon());
  for (const chunk of chunks) {
    await new Promise<void>((resolve) => terminal.write(chunk, resolve));
  }
  const cursorX = terminal.buffer.active.cursorX;
  terminal.dispose();
  return cursorX;
}

describe("xterm grapheme width profile", () => {
  it.each([
    ["😀", 2],
    ["🫠", 2],
    ["☰", 2],
    ["·", 1],
  ])("uses the Runner rune width for %s", async (value, width) => {
    expect(await cursorAfter([value])).toBe(width);
  });

  it.each([
    { name: "ZWJ", value: "👩‍💻", chunks: ["👩", "\u200D", "💻"], width: 2 },
    { name: "flag", value: "🇺🇸", chunks: ["🇺", "🇸"], width: 2 },
    { name: "VS16", value: "❤️", chunks: ["❤", "\uFE0F"], width: 2 },
    { name: "skin modifier", value: "👍🏽", chunks: ["👍", "🏽"], width: 2 },
    { name: "VS16 ZWJ", value: "🏳️‍🌈", chunks: ["🏳", "\uFE0F", "\u200D", "🌈"], width: 2 },
    { name: "keycap", value: "1️⃣", chunks: ["1", "\uFE0F", "\u20E3"], width: 2 },
  ])("keeps $name cursor width stable across write chunks", async ({ value, chunks, width }) => {
    expect(await cursorAfter([value])).toBe(width);
    expect(await cursorAfter(chunks)).toBe(width);
  });

  it("keeps a combining mark attached to its base", async () => {
    expect(await cursorAfter(["a", "\u0301"])).toBe(1);
  });

  it("restores the provider that was active before activation", () => {
    const terminal = new Terminal({ allowProposedApi: true });
    terminal.unicode.register({
      version: "preexisting",
      wcwidth: () => 1,
      charProperties: () => 2,
    });
    terminal.unicode.activeVersion = "preexisting";
    const addon = new RunnerUnicodeGraphemesAddon();

    addon.activate(terminal);
    expect(terminal.unicode.activeVersion).toBe(RUNNER_UNICODE_VERSION);

    addon.dispose();
    expect(terminal.unicode.activeVersion).toBe("preexisting");
    terminal.dispose();
  });

  it.each([
    ["zero", ZERO_WIDTH_RANGES],
    ["double", DOUBLE_WIDTH_RANGES],
  ])("keeps the generated %s-width ranges sorted and disjoint", (_name, ranges) => {
    expect(ranges.length % 2).toBe(0);
    for (let index = 0; index < ranges.length; index += 2) {
      expect(ranges[index]).toBeLessThanOrEqual(ranges[index + 1]);
      if (index > 0) expect(ranges[index - 1]).toBeLessThan(ranges[index]);
    }
  });

  it("matches the Runner fixture rune widths", () => {
    expect([
      runnerRuneWidth("😀".codePointAt(0)!),
      runnerRuneWidth("🫠".codePointAt(0)!),
      runnerRuneWidth("☰".codePointAt(0)!),
      runnerRuneWidth("❤".codePointAt(0)!),
      runnerRuneWidth("界".codePointAt(0)!),
      runnerRuneWidth("·".codePointAt(0)!),
      runnerRuneWidth(0x0301),
      runnerRuneWidth(0xFE0F),
    ]).toEqual([2, 2, 2, 1, 2, 1, 0, 0]);
  });
});
