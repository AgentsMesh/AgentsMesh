import { describe, expect, it } from "vitest";

import { RelayPodSession } from "../relayPodSession";

describe("RelayPodSession", () => {
  it("rejects duplicate subscription IDs without replacing the owner token", () => {
    const session = new RelayPodSession("pod-1");
    const token = session.beginSubscription("terminal");

    expect(() => session.beginSubscription("terminal")).toThrowError(
      expect.objectContaining({ name: "InvalidStateError" }),
    );
    expect(session.ownsSubscription(token)).toBe(true);
  });

  it("accepts mutations only from the current token and generation", () => {
    const session = new RelayPodSession("pod-1");
    const first = session.beginSubscription("terminal");
    const impostor = { ...first };

    expect(session.ownsSubscription(impostor)).toBe(false);
    expect(session.markManagerStarted(impostor)).toBe(false);
    expect(session.markSubscriptionReady(impostor)).toBe(false);
    expect(session.removeSubscription("missing", first)).toBe(false);
    expect(session.removeSubscription("terminal", impostor)).toBe(false);

    expect(session.markManagerStarted(first)).toBe(true);
    expect(session.markSubscriptionReady(first)).toBe(true);
    expect(session.hasStartedConnection()).toBe(true);
    const peer = session.beginSubscription("acp");
    expect(session.removeSubscription("terminal", first)).toBe(true);
    expect(session.ownsSubscription(peer)).toBe(true);
    expect(session.removeSubscription("acp", peer)).toBe(true);

    const replacement = session.beginSubscription("terminal");
    expect(replacement.generation).toBeGreaterThan(first.generation);
    expect(session.ownsSubscription(first)).toBe(false);
    expect(session.markManagerStarted(first)).toBe(false);
    expect(session.markSubscriptionReady(first)).toBe(false);
  });

  it("queues the latest resize until transport and every baseline are ready", () => {
    const session = new RelayPodSession("pod-1");
    const terminal = session.beginSubscription("terminal");
    const acp = session.beginSubscription("acp");

    expect(session.scheduleResize({ cols: 80, rows: 24, force: true })).toBeUndefined();
    expect(session.scheduleResize({ cols: 120, rows: 42, force: false })).toBeUndefined();
    expect(session.takeFlushableResize()).toBeUndefined();

    session.setDataReady(true);
    expect(session.markSubscriptionReady(terminal)).toBe(true);
    expect(session.takeFlushableResize()).toBeUndefined();
    expect(session.markSubscriptionReady(acp)).toBe(true);
    expect(session.takeFlushableResize()).toEqual({ cols: 120, rows: 42, force: true });
    expect(session.takeFlushableResize()).toBeUndefined();
  });

  it("dispatches immediately only while transport and all subscribers are ready", () => {
    const session = new RelayPodSession("pod-1");
    const token = session.beginSubscription("terminal");
    session.setDataReady(true);
    session.markSubscriptionReady(token);

    expect(session.scheduleResize({ cols: 100, rows: 30, force: false })).toEqual({
      cols: 100,
      rows: 30,
      force: false,
    });

    const second = session.beginSubscription("acp");
    expect(session.scheduleResize({ cols: 101, rows: 31, force: false })).toBeUndefined();
    session.markSubscriptionReady(second);
    expect(session.takeFlushableResize()).toEqual({ cols: 101, rows: 31, force: false });
  });

  it("defensively drops a queued resize from a stale internal generation", () => {
    const session = new RelayPodSession("pod-1");
    const token = session.beginSubscription("terminal");
    session.scheduleResize({ cols: 120, rows: 40, force: true });

    const internals = session as unknown as {
      pendingResize: { generation: number; request: unknown };
    };
    internals.pendingResize.generation = token.generation - 1;
    session.setDataReady(true);
    session.markSubscriptionReady(token);

    expect(session.takeFlushableResize()).toBeUndefined();
  });

  it("preserves active work but revokes data readiness on a late disconnect", () => {
    const session = new RelayPodSession("pod-1");
    const token = session.beginSubscription("terminal");
    session.markManagerStarted(token);
    session.markSubscriptionReady(token);
    session.setDataReady(true);

    expect(session.handlePodDisconnected()).toBe(true);
    expect(session.ownsSubscription(token)).toBe(true);
    expect(session.scheduleResize({ cols: 90, rows: 28, force: false })).toBeUndefined();

    session.setDataReady(true);
    expect(session.takeFlushableResize()).toEqual({ cols: 90, rows: 28, force: false });
  });

  it("retires an idle generation on disconnect and drops resize state with its last owner", () => {
    const session = new RelayPodSession("pod-1");
    expect(session.scheduleResize({ cols: 80, rows: 24, force: false })).toBeUndefined();
    expect(session.handlePodDisconnected()).toBe(false);

    const token = session.beginSubscription("terminal");
    session.scheduleResize({ cols: 140, rows: 50, force: true });
    expect(session.removeSubscription("terminal")).toBe(true);
    expect(session.takeFlushableResize()).toBeUndefined();
    expect(session.hasStartedConnection()).toBe(false);
    expect(session.ownsSubscription(token)).toBe(false);
  });

  it("clear invalidates all owners and starts a fresh generation", () => {
    const session = new RelayPodSession("pod-1");
    const old = session.beginSubscription("terminal");
    session.markManagerStarted(old);
    session.markSubscriptionReady(old);
    session.setDataReady(true);

    session.clear();

    expect(session.ownsSubscription(old)).toBe(false);
    expect(session.hasStartedConnection()).toBe(false);
    expect(session.takeFlushableResize()).toBeUndefined();
    const next = session.beginSubscription("terminal");
    expect(next.generation).toBeGreaterThan(old.generation);
  });
});
