import { describe, expect, it } from "vitest";

import { RelayPodRegistry } from "../relayPodRegistry";

describe("RelayPodRegistry", () => {
  it("creates one stable session per pod", () => {
    const registry = new RelayPodRegistry();

    expect(registry.get("pod-1")).toBeUndefined();
    const first = registry.getOrCreate("pod-1");
    expect(registry.getOrCreate("pod-1")).toBe(first);
    expect(registry.get("pod-1")).toBe(first);
    expect(registry.getOrCreate("pod-2")).not.toBe(first);
  });

  it("keeps a session with active work across a late disconnect", () => {
    const registry = new RelayPodRegistry();
    const session = registry.getOrCreate("pod-1");
    const token = session.beginSubscription("terminal");

    expect(registry.handleDisconnected("pod-1")).toBe(true);
    expect(registry.get("pod-1")).toBe(session);
    expect(session.ownsSubscription(token)).toBe(true);
  });

  it("removes idle and unknown sessions on disconnect", () => {
    const registry = new RelayPodRegistry();
    registry.getOrCreate("idle");

    expect(registry.handleDisconnected("idle")).toBe(false);
    expect(registry.get("idle")).toBeUndefined();
    expect(registry.handleDisconnected("unknown")).toBe(false);
  });

  it("clear invalidates one session and tolerates an unknown pod", () => {
    const registry = new RelayPodRegistry();
    const session = registry.getOrCreate("pod-1");
    const token = session.beginSubscription("terminal");

    registry.clear("unknown");
    registry.clear("pod-1");

    expect(registry.get("pod-1")).toBeUndefined();
    expect(session.ownsSubscription(token)).toBe(false);
  });

  it("clearAll invalidates every registered session", () => {
    const registry = new RelayPodRegistry();
    const one = registry.getOrCreate("pod-1");
    const two = registry.getOrCreate("pod-2");
    const oneToken = one.beginSubscription("terminal");
    const twoToken = two.beginSubscription("terminal");

    registry.clearAll();

    expect(registry.get("pod-1")).toBeUndefined();
    expect(registry.get("pod-2")).toBeUndefined();
    expect(one.ownsSubscription(oneToken)).toBe(false);
    expect(two.ownsSubscription(twoToken)).toBe(false);
    registry.clearAll();
  });
});
