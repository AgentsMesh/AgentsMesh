import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => {
  const manager = {
    subscribe: vi.fn().mockResolvedValue(undefined),
    unsubscribe: vi.fn().mockResolvedValue(undefined),
    send: vi.fn().mockResolvedValue(undefined),
    send_resize: vi.fn().mockResolvedValue(undefined),
    force_resize: vi.fn().mockResolvedValue(undefined),
    send_acp_command: vi.fn().mockResolvedValue(undefined),
    disconnect: vi.fn().mockResolvedValue(undefined),
    disconnect_all: vi.fn().mockResolvedValue(undefined),
    on_status_change: vi.fn().mockResolvedValue(undefined),
    on_acp_message: vi.fn().mockResolvedValue(undefined),
    on_pod_disconnected: vi.fn().mockResolvedValue(undefined),
  };
  return {
    manager,
    selectRelayEndpoint: vi.fn().mockResolvedValue({
      url: "wss://relay.example",
      token: "token",
    }),
  };
});

vi.mock("@agentsmesh/service-runtime", () => ({
  getRelayManager: () => mocks.manager,
}));
vi.mock("../relayEndpointSelection", () => ({
  selectRelayEndpoint: mocks.selectRelayEndpoint,
}));

import { relayPool } from "../relayConnection";

type Endpoint = { url: string; token: string };
type StatusRaw = { status: string; runnerDisconnected: boolean };

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((done, fail) => {
    resolve = done;
    reject = fail;
  });
  return { promise, resolve, reject };
}

function statusCallback(podKey: string): (raw: StatusRaw) => void {
  const call = mocks.manager.on_status_change.mock.calls.findLast(
    ([key]) => key === podKey,
  );
  return call![1] as (raw: StatusRaw) => void;
}

describe("relayConnection static adapter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.manager.subscribe.mockResolvedValue(undefined);
    mocks.selectRelayEndpoint.mockResolvedValue({
      url: "wss://relay.example",
      token: "token",
    });
  });

  it("covers the public adapter surface and ready resize delivery", async () => {
    const podKey = "direct-surface";
    const onMessage = vi.fn();
    const status = vi.fn();
    const acp = vi.fn();
    const offStatus = relayPool.onStatusChange(podKey, status);
    const offAcp = relayPool.onAcpMessage(podKey, acp);
    const handle = await relayPool.subscribe(podKey, "terminal", onMessage);

    expect(mocks.selectRelayEndpoint).toHaveBeenCalledWith(podKey);
    expect(mocks.manager.subscribe).toHaveBeenCalledWith(
      podKey,
      "terminal",
      "wss://relay.example",
      "token",
      onMessage,
    );
    statusCallback(podKey)({ status: "connected", runnerDisconnected: true });
    expect(relayPool.getStatus(podKey)).toBe("connected");
    expect(relayPool.isConnected(podKey)).toBe(true);
    expect(relayPool.isRunnerDisconnected(podKey)).toBe(true);

    relayPool.sendResize(podKey, 80, 24);
    relayPool.forceResize(podKey, 100, 40);
    handle.send("input");
    relayPool.sendAcpCommand(podKey, { type: "prompt", text: "hello" });
    expect(mocks.manager.send_resize).toHaveBeenCalledWith(podKey, 80, 24);
    expect(mocks.manager.force_resize).toHaveBeenCalledWith(podKey, 100, 40);
    expect(mocks.manager.send).toHaveBeenCalledWith(podKey, "input");
    expect(mocks.manager.send_acp_command).toHaveBeenCalledWith(
      podKey,
      JSON.stringify({ type: "prompt", text: "hello" }),
    );
    expect(relayPool.getPodSize()).toBeUndefined();

    const acpCallback = mocks.manager.on_acp_message.mock.calls.at(-1)![1] as (
      type: number,
      payload: unknown,
    ) => void;
    acpCallback(11, { type: "chunk" });
    expect(acp).toHaveBeenCalledWith(11, { type: "chunk" });

    offStatus();
    offAcp();
    handle.unsubscribe();
    handle.unsubscribe();
    relayPool.disconnect(podKey);
    relayPool.disconnectAll();
    window.dispatchEvent(new Event("beforeunload"));
    expect(mocks.manager.unsubscribe).toHaveBeenCalledTimes(1);
    expect(mocks.manager.disconnect).toHaveBeenCalledWith(podKey);
    expect(mocks.manager.disconnect_all).toHaveBeenCalledTimes(2);
  });

  it("queues a sticky force resize until the baseline and transport are ready", async () => {
    const podKey = "direct-queued-resize";
    const baseline = deferred<void>();
    mocks.manager.subscribe.mockReturnValueOnce(baseline.promise);

    const pending = relayPool.subscribe(podKey, "terminal", vi.fn());
    await vi.waitFor(() => expect(mocks.manager.subscribe).toHaveBeenCalled());
    relayPool.forceResize(podKey, 90, 30);
    relayPool.sendResize(podKey, 120, 44);
    relayPool.sendResize(podKey, 0, 24);
    relayPool.forceResize(podKey, 80, 0);
    expect(mocks.manager.force_resize).not.toHaveBeenCalled();

    statusCallback(podKey)({ status: "connected", runnerDisconnected: false });
    baseline.resolve();
    await pending;
    expect(mocks.manager.force_resize).toHaveBeenCalledExactlyOnceWith(
      podKey,
      120,
      44,
    );
  });

  it("rejects a signal that was already aborted without selecting an endpoint", async () => {
    const controller = new AbortController();
    controller.abort();

    await expect(
      relayPool.subscribe("direct-pre-abort", "terminal", vi.fn(), controller.signal),
    ).rejects.toMatchObject({ name: "AbortError" });
    expect(mocks.selectRelayEndpoint).not.toHaveBeenCalled();
    expect(mocks.manager.subscribe).not.toHaveBeenCalled();
  });

  it("aborts endpoint selection immediately and ignores its eventual completion", async () => {
    const endpoint = deferred<Endpoint>();
    mocks.selectRelayEndpoint.mockReturnValueOnce(endpoint.promise);
    const controller = new AbortController();
    const pending = relayPool.subscribe(
      "direct-endpoint-abort",
      "terminal",
      vi.fn(),
      controller.signal,
    );

    controller.abort();
    await expect(pending).rejects.toMatchObject({ name: "AbortError" });
    endpoint.resolve({ url: "wss://late", token: "late" });
    await Promise.resolve();
    expect(mocks.manager.subscribe).not.toHaveBeenCalled();
    expect(mocks.manager.unsubscribe).not.toHaveBeenCalled();
  });

  it("rejects direct cancellation while endpoint selection is pending", async () => {
    const endpoint = deferred<Endpoint>();
    mocks.selectRelayEndpoint.mockReturnValueOnce(endpoint.promise);
    const podKey = "direct-endpoint-cancel";
    const pending = relayPool.subscribe(podKey, "terminal", vi.fn());

    relayPool.unsubscribe(podKey, "terminal");
    endpoint.resolve({ url: "wss://relay", token: "token" });

    await expect(pending).rejects.toMatchObject({ name: "AbortError" });
    expect(mocks.manager.subscribe).not.toHaveBeenCalled();
    expect(mocks.manager.unsubscribe).toHaveBeenCalledWith(podKey, "terminal");
  });

  it("rejects a subscriber superseded while its manager baseline is pending", async () => {
    const baseline = deferred<void>();
    mocks.manager.subscribe.mockReturnValueOnce(baseline.promise);
    const podKey = "direct-superseded";
    const pending = relayPool.subscribe(podKey, "terminal", vi.fn());
    await vi.waitFor(() => expect(mocks.manager.subscribe).toHaveBeenCalled());

    relayPool.unsubscribe(podKey, "terminal");
    baseline.resolve();

    await expect(pending).rejects.toMatchObject({ name: "AbortError" });
  });

  it("unsubscribes a manager attempt aborted during its baseline", async () => {
    const baseline = deferred<void>();
    mocks.manager.subscribe.mockReturnValueOnce(baseline.promise);
    const controller = new AbortController();
    const podKey = "direct-baseline-abort";
    const pending = relayPool.subscribe(
      podKey,
      "terminal",
      vi.fn(),
      controller.signal,
    );
    await vi.waitFor(() => expect(mocks.manager.subscribe).toHaveBeenCalled());

    controller.abort();
    baseline.resolve();

    await expect(pending).rejects.toMatchObject({ name: "AbortError" });
    expect(mocks.manager.unsubscribe).toHaveBeenCalledWith(podKey, "terminal");
  });

  it("cleans up and propagates endpoint and manager failures", async () => {
    const endpointError = new Error("endpoint failed");
    mocks.selectRelayEndpoint.mockRejectedValueOnce(endpointError);
    await expect(
      relayPool.subscribe("direct-endpoint-failure", "terminal", vi.fn()),
    ).rejects.toBe(endpointError);

    const managerError = new Error("manager failed");
    mocks.manager.subscribe.mockRejectedValueOnce(managerError);
    await expect(
      relayPool.subscribe("direct-manager-failure", "terminal", vi.fn()),
    ).rejects.toBe(managerError);
  });

  it("tolerates unknown unsubscribe and ignores resize without an active owner", () => {
    relayPool.unsubscribe("direct-unknown", "missing");
    relayPool.sendResize("direct-unknown", 80, 24);
    relayPool.forceResize("direct-unknown", 100, 40);

    expect(mocks.manager.unsubscribe).toHaveBeenCalledWith("direct-unknown", "missing");
    expect(mocks.manager.send_resize).not.toHaveBeenCalled();
    expect(mocks.manager.force_resize).not.toHaveBeenCalled();
    expect(relayPool.getStatus("direct-unknown")).toBe("none");
    expect(relayPool.isConnected("direct-unknown")).toBe(false);
    expect(relayPool.isRunnerDisconnected("direct-unknown")).toBe(false);
  });

  it("reuses the production singleton and replaces the development singleton", async () => {
    vi.stubEnv("NODE_ENV", "production");
    vi.resetModules();
    const production = await import("../relayConnection");
    expect(production.relayPool).toBe(relayPool);

    vi.stubEnv("NODE_ENV", "development");
    vi.resetModules();
    const development = await import("../relayConnection");
    expect(development.relayPool).not.toBe(relayPool);
    expect(mocks.manager.disconnect_all).toHaveBeenCalledOnce();
    vi.unstubAllEnvs();
  });
});
