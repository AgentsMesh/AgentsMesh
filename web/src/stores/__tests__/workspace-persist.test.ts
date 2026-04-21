import { describe, it, expect, beforeEach } from "vitest";
import { useWorkspaceStore } from "../workspace";

// Regression: zustand persist rehydrate used to shallow-merge a corrupted
// localStorage snapshot over defaults, so `panes: null` (from a pre-v4
// write) would shadow the `[]` initial value. Every subsequent `.find()`
// in the store then NPE'd during a pod:status_changed realtime event.
// The persist config now forces `panes` to an array.

const KEY = "agentsmesh-workspace";

function seedStorage(raw: unknown) {
  localStorage.setItem(KEY, JSON.stringify({
    version: 4,
    state: raw,
  }));
}

describe("workspaceStore · persist merge", () => {
  beforeEach(() => {
    localStorage.clear();
    // Force re-create store from scratch for each test.
    useWorkspaceStore.persist.clearStorage?.();
  });

  it("coerces null-panes snapshot to []", async () => {
    seedStorage({ panes: null, splitTree: null });
    await useWorkspaceStore.persist.rehydrate?.();

    const panes = useWorkspaceStore.getState().panes;
    expect(Array.isArray(panes)).toBe(true);
    expect(panes).toHaveLength(0);

    // The canonical NPE path that the bug produced.
    expect(() => useWorkspaceStore.getState().removePaneByPodKey("whatever")).not.toThrow();
  });

  it("coerces missing-panes snapshot to []", async () => {
    seedStorage({ splitTree: null });
    await useWorkspaceStore.persist.rehydrate?.();

    expect(useWorkspaceStore.getState().panes).toEqual([]);
  });

  it("preserves a valid panes array", async () => {
    const panes = [{ id: "a", podKey: "1-ws-abc" }];
    seedStorage({ panes, splitTree: null });
    await useWorkspaceStore.persist.rehydrate?.();

    expect(useWorkspaceStore.getState().panes).toEqual(panes);
  });
});
