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

describe("relayConnection delivery and fan-out", () => {
  beforeEach(async () => {
    pool = await resetRelayConnectionPool();
  });

  describe("input / resize delivery", () => {
    it("drops a resize that has no active subscription generation", async () => {
      pool.forceResize("pod-1", 88, 28);

      const subscription = pool.subscribe("pod-1", "sub-1", vi.fn());
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalled());
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await subscription;

      expect(mgr.force_resize).not.toHaveBeenCalled();
    });

    it("queues the latest initial size until a subscriber baseline is ready", async () => {
      const subscription = pool.subscribe("pod-1", "sub-1", vi.fn());
      pool.send("pod-1", "data");
      pool.sendResize("pod-1", 80, 24);
      pool.forceResize("pod-1", 100, 40);
      pool.sendResize("pod-1", 0, 24);
      pool.forceResize("pod-1", 80, 0);

      expect(mgr.send).toHaveBeenCalledWith("pod-1", "data");
      expect(mgr.send_resize).not.toHaveBeenCalled();
      expect(mgr.force_resize).not.toHaveBeenCalled();

      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalled());
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await subscription;
      expect(mgr.force_resize).toHaveBeenCalledExactlyOnceWith("pod-1", 100, 40);
    });

    it("keeps force sticky when a normal resize supplies the latest dimensions", async () => {
      const subscription = pool.subscribe("pod-1", "sub-1", vi.fn());
      pool.forceResize("pod-1", 90, 30);
      pool.sendResize("pod-1", 120, 44);

      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalled());
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await subscription;

      expect(mgr.send_resize).not.toHaveBeenCalled();
      expect(mgr.force_resize).toHaveBeenCalledExactlyOnceWith("pod-1", 120, 44);
    });

    it("does not lose a resize while endpoint selection is unresolved", async () => {
      let resolveEndpoint!: (value: never) => void;
      getPodConnectionMock().mockImplementationOnce(
        () => new Promise((resolve) => { resolveEndpoint = resolve as (value: never) => void; }),
      );

      const subscription = pool.subscribe("pod-1", "sub-1", vi.fn());
      pool.forceResize("pod-1", 96, 31);
      expect(mgr.force_resize).not.toHaveBeenCalled();

      resolveEndpoint({
        relay_url: "wss://relay.example.com",
        token: "test-token",
        pod_key: "pod-1",
      } as never);
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalled());
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await subscription;
      expect(mgr.force_resize).toHaveBeenCalledExactlyOnceWith("pod-1", 96, 31);
    });

    it("does not flush an aborted endpoint attempt's resize into the next subscription", async () => {
      let resolveFirstEndpoint!: (value: never) => void;
      getPodConnectionMock().mockImplementationOnce(
        () => new Promise((resolve) => {
          resolveFirstEndpoint = resolve as (value: never) => void;
        }),
      );
      const firstController = new AbortController();
      const first = pool.subscribe("pod-1", "sub-a", vi.fn(), firstController.signal);
      const firstRejected = expect(first).rejects.toMatchObject({ name: "AbortError" });
      pool.forceResize("pod-1", 91, 30);

      firstController.abort();
      const second = pool.subscribe("pod-1", "sub-b", vi.fn());
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalledTimes(1));
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await second;

      expect(mgr.force_resize).not.toHaveBeenCalled();
      expect(mgr.send_resize).not.toHaveBeenCalled();

      resolveFirstEndpoint({
        relay_url: "wss://relay.example.com",
        token: "test-token",
        pod_key: "pod-1",
      } as never);
      await firstRejected;
      expect(mgr.subscribe).toHaveBeenCalledTimes(1);
    });

    it("flushes only the replacement resize from a new subscription generation", async () => {
      let resolveFirstEndpoint!: (value: never) => void;
      getPodConnectionMock().mockImplementationOnce(
        () => new Promise((resolve) => {
          resolveFirstEndpoint = resolve as (value: never) => void;
        }),
      );
      const firstController = new AbortController();
      const first = pool.subscribe("pod-1", "sub-a", vi.fn(), firstController.signal);
      const firstRejected = expect(first).rejects.toMatchObject({ name: "AbortError" });
      pool.forceResize("pod-1", 91, 30);
      firstController.abort();

      const second = pool.subscribe("pod-1", "sub-b", vi.fn());
      pool.sendResize("pod-1", 132, 48);
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalledTimes(1));
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await second;

      expect(mgr.force_resize).not.toHaveBeenCalled();
      expect(mgr.send_resize).toHaveBeenCalledExactlyOnceWith("pod-1", 132, 48);

      resolveFirstEndpoint({
        relay_url: "wss://relay.example.com",
        token: "test-token",
        pod_key: "pod-1",
      } as never);
      await firstRejected;
      expect(mgr.subscribe).toHaveBeenCalledTimes(1);
    });

    it("keeps resize queued while a second subscriber is awaiting its baseline", async () => {
      const first = pool.subscribe("pod-1", "sub-1", vi.fn());
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalledTimes(1));
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await first;
      mgr.force_resize.mockClear();

      let resolveSecond!: () => void;
      mgr.subscribe.mockImplementationOnce(
        () => new Promise<void>((resolve) => { resolveSecond = resolve; }),
      );
      const second = pool.subscribe("pod-1", "sub-2", vi.fn());
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalledTimes(2));
      pool.forceResize("pod-1", 110, 42);
      expect(mgr.force_resize).not.toHaveBeenCalled();

      resolveSecond();
      await second;
      expect(mgr.force_resize).toHaveBeenCalledExactlyOnceWith("pod-1", 110, 42);
    });

    it("flushes a queued resize when a pending sibling aborts and a ready subscriber remains", async () => {
      const first = pool.subscribe("pod-1", "sub-1", vi.fn());
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalledTimes(1));
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await first;
      mgr.force_resize.mockClear();

      let resolveSecond!: () => void;
      mgr.subscribe.mockImplementationOnce(
        () => new Promise<void>((resolve) => { resolveSecond = resolve; }),
      );
      const controller = new AbortController();
      const second = pool.subscribe("pod-1", "sub-2", vi.fn(), controller.signal);
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalledTimes(2));
      pool.forceResize("pod-1", 120, 44);
      controller.abort();
      expect(mgr.force_resize).toHaveBeenCalledExactlyOnceWith("pod-1", 120, 44);
      expect(mgr.unsubscribe.mock.invocationCallOrder.at(-1)).toBeLessThan(
        mgr.force_resize.mock.invocationCallOrder.at(-1)!,
      );

      resolveSecond();
      await expect(second).rejects.toMatchObject({ name: "AbortError" });
    });

    it("queues during reconnect and flushes only after data is ready again", async () => {
      const subscription = pool.subscribe("pod-1", "sub-1", vi.fn());
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalled());
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await subscription;
      mgr.send_resize.mockClear();

      lastStatusCb()({ status: "connecting", runnerDisconnected: false });
      pool.sendResize("pod-1", 101, 33);
      expect(mgr.send_resize).not.toHaveBeenCalled();
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      expect(mgr.send_resize).toHaveBeenCalledExactlyOnceWith("pod-1", 101, 33);
    });

    it("delegates immediately once the pod and all subscribers are ready", async () => {
      const subscription = pool.subscribe("pod-1", "sub-1", vi.fn());
      await vi.waitFor(() => expect(mgr.subscribe).toHaveBeenCalled());
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      await subscription;

      pool.sendResize("pod-1", 80, 24);
      pool.forceResize("pod-1", 100, 40);
      expect(mgr.send_resize).toHaveBeenCalledExactlyOnceWith("pod-1", 80, 24);
      expect(mgr.force_resize).toHaveBeenCalledExactlyOnceWith("pod-1", 100, 40);
    });

    it("sendAcpCommand JSON-encodes the command for the string-typed manager", () => {
      pool.sendAcpCommand("pod-1", { type: "prompt", prompt: "hi" });
      expect(mgr.send_acp_command).toHaveBeenCalledWith(
        "pod-1", JSON.stringify({ type: "prompt", prompt: "hi" }),
      );
    });
  });

  describe("status fan-out + 'none' baseline", () => {
    it("emits 'none' immediately for an unknown pod and maps pre-connect 'disconnected' to 'none'", () => {
      const listener = vi.fn();
      pool.onStatusChange("pod-1", listener);
      expect(listener).toHaveBeenCalledWith({ status: "none", runnerDisconnected: false });

      lastStatusCb()({ status: "disconnected", runnerDisconnected: false });
      expect(listener).toHaveBeenLastCalledWith({ status: "none", runnerDisconnected: false });
    });

    it("passes through real statuses once subscribed and updates isConnected/getStatus", async () => {
      const listener = vi.fn();
      pool.onStatusChange("pod-1", listener);
      await pool.subscribe("pod-1", "sub-1", vi.fn());

      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      expect(listener).toHaveBeenLastCalledWith({ status: "connected", runnerDisconnected: false });
      expect(pool.isConnected("pod-1")).toBe(true);
      expect(pool.getStatus("pod-1")).toBe("connected");

      lastStatusCb()({ status: "disconnected", runnerDisconnected: true });
      expect(pool.isConnected("pod-1")).toBe(false);
      expect(pool.isRunnerDisconnected("pod-1")).toBe(true);
    });

    it("stops notifying a removed listener", () => {
      const listener = vi.fn();
      const off = pool.onStatusChange("pod-1", listener);
      listener.mockClear();
      off();
      lastStatusCb()({ status: "connected", runnerDisconnected: false });
      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe("acp fan-out", () => {
    it("routes manager ACP messages to registered listeners until removed", () => {
      const listener = vi.fn();
      const off = pool.onAcpMessage("pod-1", listener);
      lastAcpCb()(0x0b, { type: "contentChunk" });
      expect(listener).toHaveBeenCalledWith(0x0b, { type: "contentChunk" });

      off();
      listener.mockClear();
      lastAcpCb()(0x0b, { type: "more" });
      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe("defaults for unknown pods", () => {
    it("getStatus/isConnected/isRunnerDisconnected return safe defaults", () => {
      expect(pool.getStatus("unknown")).toBe("none");
      expect(pool.isConnected("unknown")).toBe(false);
      expect(pool.isRunnerDisconnected("unknown")).toBe(false);
    });

    it("disconnect / disconnectAll delegate to the manager", async () => {
      await pool.subscribe("pod-1", "sub-1", vi.fn());
      pool.disconnect("pod-1");
      pool.disconnectAll();
      expect(mgr.disconnect).toHaveBeenCalledWith("pod-1");
      expect(mgr.disconnect_all).toHaveBeenCalled();
    });
  });
});
