import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  getPodConnectionMock,
  lastAcpCb,
  lastStatusCb,
  mgr,
  podDisconnectedCb,
  resetRelayConnectionPool,
} from "./relayConnection.fixture.test";

let pool: Awaited<ReturnType<typeof resetRelayConnectionPool>>;

describe("relayConnection subscription generations", () => {
  beforeEach(async () => {
    pool = await resetRelayConnectionPool();
  });

  describe("subscribe", () => {
    it("selects the endpoint then delegates to the manager and returns a handle", async () => {
      const onMessage = vi.fn();
      const handle = await pool.subscribe("pod-1", "sub-1", onMessage);

      expect(mgr.subscribe).toHaveBeenCalledWith(
        "pod-1", "sub-1", "wss://relay.example.com", "test-token", onMessage,
      );
      expect(handle).toHaveProperty("send");
      expect(handle).toHaveProperty("unsubscribe");
    });

    it("registers exactly one upstream status listener per pod", async () => {
      await pool.subscribe("pod-1", "sub-1", vi.fn());
      pool.onStatusChange("pod-1", vi.fn());
      await pool.subscribe("pod-1", "sub-2", vi.fn());

      expect(mgr.on_status_change).toHaveBeenCalledTimes(1);
    });

    it("handle.send / handle.unsubscribe delegate to the manager", async () => {
      const handle = await pool.subscribe("pod-1", "sub-1", vi.fn());
      handle.send("x");
      handle.unsubscribe();
      expect(mgr.send).toHaveBeenCalledWith("pod-1", "x");
      expect(mgr.unsubscribe).toHaveBeenCalledWith("pod-1", "sub-1");
    });

    it("rejects a duplicate id without replacing an active subscription", async () => {
      const first = await pool.subscribe("pod-1", "same", vi.fn());

      await expect(
        pool.subscribe("pod-1", "same", vi.fn()),
      ).rejects.toMatchObject({ name: "InvalidStateError" });
      expect(mgr.subscribe).toHaveBeenCalledTimes(1);

      first.unsubscribe();
    });

    it("rejects a duplicate id while the first endpoint selection is pending", async () => {
      let resolveEndpoint!: (value: never) => void;
      getPodConnectionMock().mockImplementationOnce(
        () => new Promise((resolve) => { resolveEndpoint = resolve as (value: never) => void; }),
      );

      const first = pool.subscribe("pod-1", "same", vi.fn());
      await expect(
        pool.subscribe("pod-1", "same", vi.fn()),
      ).rejects.toMatchObject({ name: "InvalidStateError" });
      expect(getPodConnectionMock()).toHaveBeenCalledTimes(1);
      expect(mgr.subscribe).not.toHaveBeenCalled();

      resolveEndpoint({
        relay_url: "wss://relay.example.com",
        token: "test-token",
        pod_key: "pod-1",
      } as never);
      const handle = await first;
      handle.unsubscribe();
    });

    it("cancels a subscription that is still waiting for its baseline", async () => {
      let resolveSubscribe!: () => void;
      mgr.subscribe.mockImplementationOnce(
        () => new Promise<void>((resolve) => { resolveSubscribe = resolve; }),
      );
      const controller = new AbortController();
      const pending = pool.subscribe("pod-1", "pending", vi.fn(), controller.signal);
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalled());

      controller.abort();
      expect(mgr.unsubscribe).toHaveBeenCalledWith("pod-1", "pending");
      resolveSubscribe();
      await expect(pending).rejects.toMatchObject({ name: "AbortError" });
    });

    it("does not start a manager subscription after direct unsubscribe during endpoint selection", async () => {
      let resolveEndpoint!: (value: never) => void;
      getPodConnectionMock().mockImplementationOnce(
        () => new Promise((resolve) => { resolveEndpoint = resolve as (value: never) => void; }),
      );

      const pending = pool.subscribe("pod-1", "pending", vi.fn());
      pool.unsubscribe("pod-1", "pending");
      expect(mgr.unsubscribe).toHaveBeenCalledWith("pod-1", "pending");

      resolveEndpoint({
        relay_url: "wss://relay.example.com",
        token: "test-token",
        pod_key: "pod-1",
      } as never);
      await expect(pending).rejects.toMatchObject({ name: "AbortError" });
      expect(mgr.subscribe).not.toHaveBeenCalled();
    });

    it("settles immediately on abort and observes a late endpoint rejection", async () => {
      let rejectEndpoint!: (reason: unknown) => void;
      getPodConnectionMock().mockImplementationOnce(
        () => new Promise((_resolve, reject) => { rejectEndpoint = reject; }),
      );
      const controller = new AbortController();
      const pending = pool.subscribe("pod-1", "aborted", vi.fn(), controller.signal);

      controller.abort();
      await expect(pending).rejects.toMatchObject({ name: "AbortError" });
      expect(mgr.subscribe).not.toHaveBeenCalled();
      expect(mgr.unsubscribe).not.toHaveBeenCalled();

      rejectEndpoint(new Error("late endpoint failure"));
      await Promise.resolve();
      await Promise.resolve();
    });

    it("preserves a new endpoint attempt across the old driver's grace teardown", async () => {
      const listener = vi.fn();
      pool.onStatusChange("pod-1", listener);
      const old = await pool.subscribe("pod-1", "old", vi.fn());
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      old.unsubscribe();

      let resolveEndpoint!: (value: never) => void;
      getPodConnectionMock().mockImplementationOnce(
        () => new Promise((resolve) => { resolveEndpoint = resolve as (value: never) => void; }),
      );
      const next = pool.subscribe("pod-1", "next", vi.fn());
      pool.onAcpMessage("pod-1", vi.fn());
      listener.mockClear();

      podDisconnectedCb()("pod-1");

      expect(listener).toHaveBeenCalledWith({ status: "none", runnerDisconnected: false });
      expect(pool.getStatus("pod-1")).toBe("none");
      expect(pool.isConnected("pod-1")).toBe(false);
      expect(mgr.on_status_change).toHaveBeenCalledTimes(2);
      expect(mgr.on_acp_message).toHaveBeenCalledTimes(2);

      resolveEndpoint({
        relay_url: "wss://relay.example.com",
        token: "test-token",
        pod_key: "pod-1",
      } as never);
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalledTimes(2));
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await expect(next).resolves.toHaveProperty("unsubscribe");
    });

    it("preserves a ready new manager attempt across a late old-driver callback", async () => {
      const handle = await pool.subscribe("pod-1", "new-generation", vi.fn());
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      mgr.force_resize.mockClear();

      podDisconnectedCb()("pod-1");
      pool.forceResize("pod-1", 125, 45);
      expect(mgr.force_resize).not.toHaveBeenCalled();

      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      expect(mgr.force_resize).toHaveBeenCalledExactlyOnceWith("pod-1", 125, 45);

      handle.unsubscribe();
      expect(mgr.unsubscribe).toHaveBeenCalledWith("pod-1", "new-generation");
    });
  });
});
