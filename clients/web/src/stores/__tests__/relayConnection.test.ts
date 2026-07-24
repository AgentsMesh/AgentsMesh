import { describe, expect, it, vi, beforeEach } from "vitest";
import { registerServiceProvider, markServiceReady } from "@agentsmesh/service-runtime";
import { getPodConnection } from "@/lib/api/facade/podConnect";

// The adapter delegates all connection management to getRelayManager() (the
// Rust pool via WasmRelayManager / ElectronRelayManager). These tests pin the
// adapter's remaining responsibilities: endpoint delegation, the per-pod
// listener fan-out, the legacy "none" status baseline, and the sync status
// cache that isConnected()/getStatus() read. We drive getRelayManager() through
// the real service registry rather than vi.mock — the workspace package isn't
// reliably hoist-mockable, but registerServiceProvider() is the supported seam.
const mgr = {
  subscribe: vi.fn().mockResolvedValue(undefined),
  unsubscribe: vi.fn().mockResolvedValue(undefined),
  send: vi.fn().mockResolvedValue(undefined),
  send_resize: vi.fn().mockResolvedValue(undefined),
  force_resize: vi.fn().mockResolvedValue(undefined),
  send_acp_command: vi.fn().mockResolvedValue(undefined),
  disconnect: vi.fn().mockResolvedValue(undefined),
  disconnect_all: vi.fn().mockResolvedValue(undefined),
  get_status: vi.fn().mockResolvedValue("disconnected"),
  is_runner_disconnected: vi.fn().mockResolvedValue(false),
  get_pod_size: vi.fn().mockResolvedValue(null),
  on_status_change: vi.fn().mockResolvedValue(undefined),
  on_acp_message: vi.fn().mockResolvedValue(undefined),
  on_pod_disconnected: vi.fn(),
};

vi.mock("@/lib/api/facade/podConnect", () => ({
  getPodConnection: vi.fn().mockResolvedValue({
    relay_url: "wss://relay.example.com",
    token: "test-token",
    pod_key: "pod-1",
  }),
}));

type StatusRaw = { status: string; runnerDisconnected: boolean };

async function freshPool() {
  delete (globalThis as Record<string, unknown>).__relayPool;
  vi.resetModules();
  return (await import("@/stores/relayConnection")).relayPool;
}

function lastStatusCb(): (raw: StatusRaw) => void {
  return mgr.on_status_change.mock.calls.at(-1)![1] as (raw: StatusRaw) => void;
}
function lastAcpCb(): (mt: number, pl: unknown) => void {
  return mgr.on_acp_message.mock.calls.at(-1)![1] as (mt: number, pl: unknown) => void;
}
function podDisconnectedCb(): (podKey: string) => void {
  return mgr.on_pod_disconnected.mock.calls.at(-1)![0] as (podKey: string) => void;
}

describe("relayConnection adapter", () => {
  let pool: Awaited<ReturnType<typeof freshPool>>;

  beforeEach(async () => {
    vi.clearAllMocks();
    mgr.subscribe.mockResolvedValue(undefined);
    vi.mocked(getPodConnection).mockResolvedValue({
      relay_url: "wss://relay.example.com",
      token: "test-token",
      pod_key: "pod-1",
    } as never);
    registerServiceProvider({ relayManager: mgr as never });
    markServiceReady();
    pool = await freshPool();
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
      vi.mocked(getPodConnection).mockImplementationOnce(
        () => new Promise((resolve) => { resolveEndpoint = resolve as (value: never) => void; }),
      );

      const first = pool.subscribe("pod-1", "same", vi.fn());
      await expect(
        pool.subscribe("pod-1", "same", vi.fn()),
      ).rejects.toMatchObject({ name: "InvalidStateError" });
      expect(getPodConnection).toHaveBeenCalledTimes(1);
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
      vi.mocked(getPodConnection).mockImplementationOnce(
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
      vi.mocked(getPodConnection).mockImplementationOnce(
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
      vi.mocked(getPodConnection).mockImplementationOnce(
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
      vi.mocked(getPodConnection).mockImplementationOnce(
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
      vi.mocked(getPodConnection).mockImplementationOnce(
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
      vi.mocked(getPodConnection).mockImplementationOnce(
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
