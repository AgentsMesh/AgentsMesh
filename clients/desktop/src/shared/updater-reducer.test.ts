import { describe, it, expect } from "vitest";
import { reduceUpdater, INITIAL_SNAPSHOT, type UpdaterSnapshot } from "./updater-reducer";

describe("reduceUpdater", () => {
  it("INITIAL_SNAPSHOT is idle", () => {
    expect(INITIAL_SNAPSHOT).toEqual({ state: "idle", percent: 0, availableVersion: null, error: null });
  });

  describe("transitions", () => {
    it("checking resets percent + clears error", () => {
      const s = reduceUpdater(
        { state: "idle", percent: 50, availableVersion: null, error: "old" },
        { type: "checking" },
      );
      expect(s).toMatchObject({ state: "checking", percent: 0, error: null });
    });
    it("available → downloading + version + percent 0", () => {
      const s = reduceUpdater(INITIAL_SNAPSHOT, { type: "available", version: "1.2.3" });
      expect(s).toMatchObject({ state: "downloading", availableVersion: "1.2.3", percent: 0 });
    });
    it("progress keeps version, updates percent", () => {
      const prev: UpdaterSnapshot = { state: "downloading", percent: 0, availableVersion: "1.2.3", error: null };
      const s = reduceUpdater(prev, { type: "progress", percent: 42 });
      expect(s).toMatchObject({ state: "downloading", percent: 42, availableVersion: "1.2.3" });
    });
    it("downloaded → ready + version", () => {
      const s = reduceUpdater(INITIAL_SNAPSHOT, { type: "downloaded", version: "1.2.3" });
      expect(s).toMatchObject({ state: "ready", availableVersion: "1.2.3" });
    });
    it("error → error + message", () => {
      const s = reduceUpdater(INITIAL_SNAPSHOT, { type: "error", message: "boom" });
      expect(s).toMatchObject({ state: "error", error: "boom" });
    });
  });

  describe("not-available idle dedup", () => {
    it("idle + not-available returns the SAME reference (lets main skip the push)", () => {
      expect(reduceUpdater(INITIAL_SNAPSHOT, { type: "not-available" })).toBe(INITIAL_SNAPSHOT);
    });
    it("checking + not-available → idle (real transition, new object)", () => {
      const prev: UpdaterSnapshot = { state: "checking", percent: 0, availableVersion: null, error: null };
      const s = reduceUpdater(prev, { type: "not-available" });
      expect(s.state).toBe("idle");
      expect(s).not.toBe(prev);
    });
  });

  describe("ready is terminal", () => {
    const ready: UpdaterSnapshot = { state: "ready", percent: 100, availableVersion: "1.2.3", error: null };

    it.each([
      ["checking", { type: "checking" }],
      ["not-available", { type: "not-available" }],
      ["available re-advertise", { type: "available", version: "1.2.3" }],
      ["progress", { type: "progress", percent: 50 }],
      ["error blip", { type: "error", message: "blip" }],
    ] as const)("ready + %s returns SAME reference (no banner flicker)", (_label, ev) => {
      expect(reduceUpdater(ready, ev)).toBe(ready);
    });

    it("ready + downloaded (a newer build) DOES leave ready with the new version", () => {
      const s = reduceUpdater(ready, { type: "downloaded", version: "2.0.0" });
      expect(s).toMatchObject({ state: "ready", availableVersion: "2.0.0" });
      expect(s).not.toBe(ready);
    });
  });

  describe("typical lifecycle", () => {
    it("idle → checking → available → progress → downloaded ⇒ ready", () => {
      let s: UpdaterSnapshot = INITIAL_SNAPSHOT;
      s = reduceUpdater(s, { type: "checking" });
      expect(s.state).toBe("checking");
      s = reduceUpdater(s, { type: "available", version: "1.0.0" });
      expect(s.state).toBe("downloading");
      s = reduceUpdater(s, { type: "progress", percent: 50 });
      expect(s.percent).toBe(50);
      s = reduceUpdater(s, { type: "downloaded", version: "1.0.0" });
      expect(s).toMatchObject({ state: "ready", availableVersion: "1.0.0" });
    });
  });

  it("never mutates prev", () => {
    const prev: UpdaterSnapshot = { state: "idle", percent: 0, availableVersion: null, error: null };
    const copy = { ...prev };
    reduceUpdater(prev, { type: "available", version: "1.0.0" });
    expect(prev).toEqual(copy);
  });
});
