import { describe, it, expect } from "vitest";
import { getSlashQuery } from "../prompt/loopalSlashQuery";
import { buildLoopalCommands, parseSlashInput, filterCommands } from "../prompt/loopalCommands";
import { thinkingKey, THINKING_OPTIONS } from "../prompt/loopalThinking";

const t = (k: string) => k;

describe("getSlashQuery", () => {
  it("matches a leading slash token", () => {
    expect(getSlashQuery("/comp", 5)).toEqual({ query: "comp", startIndex: 0 });
  });
  it("matches a bare slash", () => {
    expect(getSlashQuery("/", 1)).toEqual({ query: "", startIndex: 0 });
  });
  it("returns null without a leading slash", () => {
    expect(getSlashQuery("hello", 5)).toBeNull();
  });
  it("closes once a space is typed (now an argument)", () => {
    expect(getSlashQuery("/goal ship", 10)).toBeNull();
  });
  it("ignores a slash mid-text", () => {
    expect(getSlashQuery("path/to", 7)).toBeNull();
  });
});

describe("buildLoopalCommands", () => {
  it("omits goal sub-commands without a goal", () => {
    const ids = buildLoopalCommands({ hasGoal: false, t }).map((c) => c.id);
    expect(ids).toContain("goal");
    expect(ids).not.toContain("goal-pause");
  });
  it("adds goal sub-commands with a goal", () => {
    const ids = buildLoopalCommands({ hasGoal: true, t }).map((c) => c.id);
    expect(ids).toContain("goal-pause");
    expect(ids).toContain("goal-clear");
  });
  it("maps /plan to loopal.mode plan", () => {
    const plan = buildLoopalCommands({ hasGoal: false, t }).find((c) => c.id === "plan");
    expect(plan?.action).toEqual({ kind: "send", subtype: "loopal.mode", payload: { mode: "plan" } });
  });
  it("routes /model to the model popover", () => {
    const model = buildLoopalCommands({ hasGoal: false, t }).find((c) => c.id === "model");
    expect(model?.action).toEqual({ kind: "popover", popover: "model" });
  });
  it("localizes hints through the translator", () => {
    const act = buildLoopalCommands({ hasGoal: false, t }).find((c) => c.id === "act");
    expect(act?.hint).toBe("commands.act");
    expect(act?.label).toBe("/act");
  });
});

describe("parseSlashInput", () => {
  const cmds = buildLoopalCommands({ hasGoal: true, t });
  it("parses a no-arg command", () => {
    expect(parseSlashInput("/compact", cmds)?.command.id).toBe("compact");
  });
  it("parses a command with an argument", () => {
    const r = parseSlashInput("/goal ship it", cmds);
    expect(r?.command.id).toBe("goal");
    expect(r?.arg).toBe("ship it");
  });
  it("returns null for an unknown command", () => {
    expect(parseSlashInput("/bogus", cmds)).toBeNull();
  });
});

describe("filterCommands", () => {
  it("filters by id prefix", () => {
    const ids = filterCommands(buildLoopalCommands({ hasGoal: false, t }), "su").map((c) => c.id);
    expect(ids).toEqual(["suspend"]);
  });
});

describe("thinkingKey", () => {
  it("round-trips effort levels", () => {
    const high = THINKING_OPTIONS.find((o) => o.key === "high")!;
    expect(thinkingKey(high.config)).toBe("high");
  });
  it("maps auto and disabled", () => {
    expect(thinkingKey(JSON.stringify({ type: "auto" }))).toBe("auto");
    expect(thinkingKey(JSON.stringify({ type: "disabled" }))).toBe("off");
  });
  it("returns null for garbage or null", () => {
    expect(thinkingKey("not json")).toBeNull();
    expect(thinkingKey(null)).toBeNull();
  });
});
