import { describe, expect, it, vi } from "vitest";
import type { WindowRegistry } from "./window_registry";
import {
  RelayOutputSubscriptions,
  type SubscriptionAttempt,
} from "./relay_output_subscriptions";

const flushOutput = () => new Promise<void>((resolve) => setImmediate(resolve));

function subject() {
  const registry = { sendTo: vi.fn() };
  return {
    registry,
    subscriptions: new RelayOutputSubscriptions(registry as unknown as WindowRegistry),
  };
}

describe("RelayOutputSubscriptions", () => {
  it("commits only the current candidate and reports the active subscriber it replaces", () => {
    const { subscriptions } = subject();
    const first = subscriptions.begin(11, "pod-1", "terminal", "attempt-1");
    const superseded = subscriptions.begin(11, "pod-1", "terminal", "attempt-2");

    expect(first.supersededCoreSubId).toBeUndefined();
    expect(superseded.supersededCoreSubId).toBe(first.coreSubId);
    expect(subscriptions.commit(first)).toEqual({ committed: false });
    expect(subscriptions.commit(superseded)).toEqual({ committed: true, replaced: undefined });

    const replacement = subscriptions.begin(11, "pod-1", "terminal", "attempt-3");
    expect(subscriptions.commit(replacement)).toEqual({
      committed: true,
      replaced: superseded.coreSubId,
    });
  });

  it("rolls back current candidates and preserves an active route", () => {
    const { subscriptions } = subject();
    const only = subscriptions.begin(11, "pod-1", "only", "attempt-1");
    expect(subscriptions.rollback(only)).toBe(true);
    expect(subscriptions.hasPod("pod-1")).toBe(false);
    expect(subscriptions.rollback(only)).toBe(false);

    const active = subscriptions.begin(11, "pod-1", "terminal", "attempt-2");
    subscriptions.commit(active);
    const candidate = subscriptions.begin(11, "pod-1", "terminal", "attempt-3");
    expect(subscriptions.rollback(candidate)).toBe(true);
    expect(subscriptions.hasPod("pod-1")).toBe(true);
  });

  it("prunes only the rolled-back route when sibling routes still exist", () => {
    const { subscriptions } = subject();
    const rolledBack = subscriptions.begin(11, "pod-1", "one", "attempt-1");
    const sibling = subscriptions.begin(11, "pod-1", "two", "attempt-2");

    expect(subscriptions.rollback(rolledBack)).toBe(true);
    expect(subscriptions.hasPod("pod-1")).toBe(true);
    expect(subscriptions.commit(sibling).committed).toBe(true);
  });

  it("removes individual, pod, and window routes without crossing ownership", () => {
    const { subscriptions } = subject();
    const first = subscriptions.begin(11, "pod-1", "one", "attempt-1");
    subscriptions.commit(first);
    const pending = subscriptions.begin(11, "pod-1", "one", "attempt-2");
    const sameWindow = subscriptions.begin(11, "pod-1", "sibling", "attempt-sibling");
    subscriptions.commit(sameWindow);
    const secondWindow = subscriptions.begin(22, "pod-1", "two", "attempt-3");
    subscriptions.commit(secondWindow);
    const secondPod = subscriptions.begin(11, "pod-2", "three", "attempt-4");
    subscriptions.commit(secondPod);
    const otherWindowOnly = subscriptions.begin(33, "pod-3", "four", "attempt-5");
    subscriptions.commit(otherWindowOnly);

    expect(subscriptions.take(99, "missing", "missing")).toBeUndefined();
    expect(subscriptions.take(11, "pod-1", "one")).toEqual({
      podKey: "pod-1",
      coreSubIds: [first.coreSubId, pending.coreSubId],
      podUnused: false,
    });
    expect(subscriptions.takePod(99, "pod-1")).toBeUndefined();
    expect(subscriptions.takeWindow(11)).toEqual([
      { podKey: "pod-1", coreSubIds: [sameWindow.coreSubId], podUnused: false },
      { podKey: "pod-2", coreSubIds: [secondPod.coreSubId], podUnused: true },
    ]);
    expect(subscriptions.takeWindow(22)).toEqual([
      { podKey: "pod-1", coreSubIds: [secondWindow.coreSubId], podUnused: true },
    ]);
    expect(subscriptions.takeWindow(33)).toEqual([
      { podKey: "pod-3", coreSubIds: [otherWindowOnly.coreSubId], podUnused: true },
    ]);
    expect(subscriptions.hasPod("pod-1")).toBe(false);
    expect(subscriptions.hasPod("pod-2")).toBe(false);
    expect(subscriptions.hasPod("pod-3")).toBe(false);
  });

  it("fans pod events out once per owning webContents", () => {
    const { subscriptions, registry } = subject();
    subscriptions.sendToPod("missing", "relay:status", { podKey: "missing", json: "{}" });
    subscriptions.begin(11, "pod-1", "one", "attempt-1");
    subscriptions.begin(11, "pod-1", "two", "attempt-2");
    subscriptions.begin(22, "pod-1", "one", "attempt-3");

    subscriptions.sendToPod("pod-1", "relay:status", {
      podKey: "pod-1",
      json: '{"status":"connected"}',
    });

    expect(registry.sendTo).toHaveBeenCalledTimes(2);
    expect(registry.sendTo).toHaveBeenCalledWith(11, "relay:status", {
      podKey: "pod-1",
      json: '{"status":"connected"}',
    });
    expect(registry.sendTo).toHaveBeenCalledWith(22, "relay:status", {
      podKey: "pod-1",
      json: '{"status":"connected"}',
    });
  });

  it("coalesces queued output and sends it only after the candidate commits", async () => {
    const { subscriptions, registry } = subject();
    const attempt = subscriptions.begin(11, "pod-1", "terminal", "attempt-1");
    const output = subscriptions.onOutput(attempt);
    output(null, [1, 2]);
    output(null, [3]);
    subscriptions.commit(attempt);

    await flushOutput();

    expect(registry.sendTo).toHaveBeenCalledExactlyOnceWith(11, "relay:output", {
      podKey: "pod-1",
      subId: "terminal",
      attemptId: "attempt-1",
      data: Uint8Array.of(1, 2, 3),
    });
  });

  it("retains candidate output across an early flush until commit", async () => {
    const { subscriptions, registry } = subject();
    const attempt = subscriptions.begin(11, "pod-1", "terminal", "attempt-1");
    subscriptions.onOutput(attempt)(null, [4, 2]);

    await flushOutput();
    expect(registry.sendTo).not.toHaveBeenCalled();

    subscriptions.commit(attempt);
    await flushOutput();
    expect(registry.sendTo).toHaveBeenCalledExactlyOnceWith(11, "relay:output", {
      podKey: "pod-1",
      subId: "terminal",
      attemptId: "attempt-1",
      data: Uint8Array.of(4, 2),
    });
  });

  it("drops output emitted after its route was removed", async () => {
    const { subscriptions, registry } = subject();
    const attempt = subscriptions.begin(11, "pod-1", "terminal", "attempt-1");
    subscriptions.take(11, "pod-1", "terminal");

    subscriptions.onOutput(attempt)(null, [9]);
    await flushOutput();

    expect(registry.sendTo).not.toHaveBeenCalled();
  });

  it("clear removes routes and queued output", async () => {
    const { subscriptions, registry } = subject();
    const attempt = subscriptions.begin(11, "pod-1", "terminal", "attempt-1");
    subscriptions.onOutput(attempt)(null, [7]);

    subscriptions.clear();
    await flushOutput();

    expect(subscriptions.hasPod("pod-1")).toBe(false);
    expect(registry.sendTo).not.toHaveBeenCalled();
  });

  it("rejects a structurally stale attempt without mutating the current route", () => {
    const { subscriptions } = subject();
    const current = subscriptions.begin(11, "pod-1", "terminal", "attempt-1");
    const stale: SubscriptionAttempt = { ...current, coreSubId: "stale" };

    expect(subscriptions.rollback(stale)).toBe(false);
    expect(subscriptions.commit(current).committed).toBe(true);
  });
});
