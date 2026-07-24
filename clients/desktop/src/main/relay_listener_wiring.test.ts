import { beforeEach, describe, expect, it, vi } from "vitest";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";
import type { RelayOutputSubscriptions } from "./relay_output_subscriptions";
import type { WindowRegistry } from "./window_registry";

type IpcHandler = (event: { sender: { id: number } }, ...args: unknown[]) => unknown;

const ipc = vi.hoisted(() => ({
  handlers: new Map<string, IpcHandler>(),
  handle: vi.fn((channel: string, handler: IpcHandler) => { ipc.handlers.set(channel, handler); }),
  removeHandler: vi.fn(),
}));

vi.mock("electron", () => ({ ipcMain: ipc }));
vi.mock("@agentsmesh/node-bridge", () => ({ logEvent: vi.fn() }));

import { setupRelayBridge } from "./relay";
import {
  RelayListenerWiring,
  type RelayPodListeners,
} from "./relay_listener_wiring";

function makeAppState() {
  return {
    relaySubscribe: vi.fn().mockImplementation((...args: unknown[]) => {
      const onBound = args[7] as (_error: unknown, generation: number) => void;
      onBound(null, 1);
      return Promise.resolve();
    }),
    relayUnsubscribe: vi.fn().mockResolvedValue(undefined),
    relayBindPodListeners: vi.fn().mockResolvedValue(0),
    relayOnPodDisconnected: vi.fn().mockResolvedValue(undefined),
    relayDisconnect: vi.fn().mockResolvedValue(undefined),
    relayDisconnectAll: vi.fn().mockResolvedValue(undefined),
    relaySend: vi.fn().mockResolvedValue(undefined),
    relaySendResize: vi.fn().mockResolvedValue(undefined),
    relayForceResize: vi.fn().mockResolvedValue(undefined),
    relaySendAcpCommand: vi.fn().mockResolvedValue(undefined),
    relayGetStatus: vi.fn().mockResolvedValue("connected"),
    relayIsRunnerDisconnected: vi.fn().mockResolvedValue(false),
    relayGetPodSize: vi.fn().mockResolvedValue([80, 24]),
  };
}

async function invoke(channel: string, wcId: number, ...args: unknown[]) {
  const handler = ipc.handlers.get(channel);
  if (!handler) throw new Error(`missing IPC handler ${channel}`);
  return handler({ sender: { id: wcId } }, ...args);
}

let nextAttempt = 0;
const subscribe = (wcId: number, podKey: string, subId: string) =>
  invoke(
    "relay:subscribe",
    wcId,
    podKey,
    subId,
    `attempt:${++nextAttempt}`,
    "wss://relay",
    "token",
  );

describe("relay listener wiring", () => {
  beforeEach(() => {
    ipc.handlers.clear();
    vi.clearAllMocks();
    nextAttempt = 0;
  });

  it("accepts listener events before bound without regressing status revision", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    appState.relaySubscribe.mockImplementationOnce((...args: unknown[]) => {
      const onStatus = args[5] as (_error: unknown, json: string) => void;
      const onAcp = args[6] as (_error: unknown, json: string) => void;
      const onBound = args[7] as (_error: unknown, generation: number) => void;
      onStatus(null, '{"generation":1,"revision":2,"status":"connected","runnerDisconnected":false}');
      for (let sequence = 0; sequence < 129; sequence += 1) {
        onAcp(null, JSON.stringify({ generation: 1, msgType: 14, payload: { sequence } }));
      }
      onBound(null, 1);
      onStatus(null, '{"generation":1,"revision":1,"status":"connecting","runnerDisconnected":false}');
      return Promise.resolve();
    });
    setupRelayBridge(appState as unknown as AppState, registry as unknown as WindowRegistry);

    await subscribe(11, "pod-1", "acp-pane");

    expect(registry.sendTo).toHaveBeenCalledWith(11, "relay:status", {
      podKey: "pod-1",
      json: '{"generation":1,"revision":2,"status":"connected","runnerDisconnected":false}',
    });
    const acpCalls = registry.sendTo.mock.calls.filter((call) => call[1] === "relay:acp");
    expect(acpCalls).toHaveLength(129);
    expect(acpCalls[128]).toEqual([11, "relay:acp", {
      podKey: "pod-1",
      json: '{"generation":1,"msgType":14,"payload":{"sequence":128}}',
    }]);
    expect(registry.sendTo.mock.calls.filter((call) => call[1] === "relay:status")).toHaveLength(1);
  });

  it("rewires listeners after the final renderer subscription is removed", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    setupRelayBridge(appState as unknown as AppState, registry as unknown as WindowRegistry);
    await subscribe(11, "pod-1", "one");
    await invoke("relay:unsubscribe", 11, "pod-1", "one");
    await subscribe(11, "pod-1", "two");

    const second = appState.relaySubscribe.mock.calls[1];
    const status = second[5] as (_error: unknown, json: string) => void;
    const acp = second[6] as (_error: unknown, json: string) => void;
    status(null, '{"generation":1,"revision":1,"status":"connected","runnerDisconnected":false}');
    acp(null, '{"generation":1,"msgType":13,"payload":{"session":{}}}');
    expect(registry.sendTo).toHaveBeenCalledWith(11, "relay:status", {
      podKey: "pod-1",
      json: '{"generation":1,"revision":1,"status":"connected","runnerDisconnected":false}',
    });
    expect(registry.sendTo).toHaveBeenCalledWith(11, "relay:acp", {
      podKey: "pod-1",
      json: '{"generation":1,"msgType":13,"payload":{"session":{}}}',
    });
  });

  it("broadcasts a final driver disconnect after the pod loses its last route", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn(), broadcast: vi.fn() };
    setupRelayBridge(appState as unknown as AppState, registry as unknown as WindowRegistry);
    await subscribe(11, "pod-1", "terminal");
    await invoke("relay:unsubscribe", 11, "pod-1", "terminal");
    const onDisconnected = appState.relayOnPodDisconnected.mock.calls[0][0] as
      (_error: unknown, eventJson: string) => void;

    onDisconnected(null, '{"podKey":"pod-1","generation":1}');

    expect(registry.broadcast).toHaveBeenCalledExactlyOnceWith(
      "relay:pod-disconnected",
      { podKey: "pod-1", generation: 1 },
    );
    expect(registry.sendTo).not.toHaveBeenCalledWith(
      expect.anything(),
      "relay:pod-disconnected",
      expect.anything(),
    );
  });

  it("rebinds when old finalize wins before a new driver publishes its generation", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    setupRelayBridge(appState as unknown as AppState, registry as unknown as WindowRegistry);
    await subscribe(11, "pod-1", "old");
    const oldSubscribe = appState.relaySubscribe.mock.calls[0];
    const oldStatus = oldSubscribe[5] as (_error: unknown, json: string) => void;
    const oldAcp = oldSubscribe[6] as (_error: unknown, json: string) => void;
    await invoke("relay:unsubscribe", 11, "pod-1", "old");
    const onDisconnected = appState.relayOnPodDisconnected.mock.calls[0][0] as
      (_error: unknown, eventJson: string) => void;

    let publishNewDriver!: () => void;
    const newDriverGate = new Promise<void>((resolve) => { publishNewDriver = resolve; });
    let signalDriverActive!: () => void;
    const driverActive = new Promise<void>((resolve) => { signalDriverActive = resolve; });
    appState.relaySubscribe.mockImplementationOnce(async (...args: unknown[]) => {
      signalDriverActive();
      await newDriverGate;
      const onBound = args[7] as (_error: unknown, generation: number) => void;
      onBound(null, 2);
    });
    let reboundStatus!: (_error: unknown, json: string) => void;
    let reboundAcp!: (_error: unknown, json: string) => void;
    appState.relayBindPodListeners.mockImplementationOnce((_podKey, status, acp) => {
      reboundStatus = status as typeof reboundStatus;
      reboundAcp = acp as typeof reboundAcp;
      return Promise.resolve(2);
    });

    const newer = subscribe(11, "pod-1", "new");
    await vi.waitFor(() => expect(appState.relaySubscribe).toHaveBeenCalledTimes(2));
    await driverActive;
    onDisconnected(null, '{"podKey":"pod-1","generation":1}');
    await vi.waitFor(() => expect(appState.relayBindPodListeners).toHaveBeenCalledTimes(1));
    publishNewDriver();
    await newer;

    reboundStatus(null, '{"generation":2,"revision":1,"status":"connected","runnerDisconnected":false}');
    reboundAcp(null, '{"generation":2,"msgType":13,"payload":{"session":{"id":"new"}}}');
    expect(registry.sendTo).toHaveBeenCalledWith(11, "relay:status", {
      podKey: "pod-1",
      json: '{"generation":2,"revision":1,"status":"connected","runnerDisconnected":false}',
    });
    expect(registry.sendTo).toHaveBeenCalledWith(11, "relay:acp", {
      podKey: "pod-1",
      json: '{"generation":2,"msgType":13,"payload":{"session":{"id":"new"}}}',
    });

    registry.sendTo.mockClear();
    onDisconnected(null, '{"podKey":"pod-1","generation":1}');
    await Promise.resolve();
    expect(appState.relayBindPodListeners).toHaveBeenCalledTimes(1);
    oldStatus(null, '{"generation":1,"revision":99,"status":"error","runnerDisconnected":true}');
    oldAcp(null, '{"generation":1,"msgType":13,"payload":{"session":{"id":"old"}}}');
    expect(registry.sendTo).not.toHaveBeenCalled();
    reboundStatus(null, '{"generation":2,"revision":2,"status":"error","runnerDisconnected":false}');
    expect(registry.sendTo).toHaveBeenCalledWith(11, "relay:status", {
      podKey: "pod-1",
      json: '{"generation":2,"revision":2,"status":"error","runnerDisconnected":false}',
    });
    reboundStatus(null, '{"generation":2,"revision":1,"status":"connecting","runnerDisconnected":false}');
    expect(registry.sendTo).toHaveBeenCalledTimes(1);
  });

  it("does not reuse a listener lease when the bridge is immediately rebuilt", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    let finishDisconnect!: () => void;
    appState.relayDisconnectAll.mockImplementationOnce(
      () => new Promise<void>((resolve) => { finishDisconnect = resolve; }),
    );

    const firstBridge = setupRelayBridge(
      appState as unknown as AppState,
      registry as unknown as WindowRegistry,
    );
    await subscribe(11, "pod-1", "old-bridge");
    const firstLeaseId = appState.relaySubscribe.mock.calls[0][8];
    firstBridge.dispose();
    setupRelayBridge(appState as unknown as AppState, registry as unknown as WindowRegistry);
    await subscribe(22, "pod-1", "new-bridge");
    const secondLeaseId = appState.relaySubscribe.mock.calls[1][8];

    expect(firstLeaseId).toMatch(/^desktop-listeners:/);
    expect(secondLeaseId).toMatch(/^desktop-listeners:/);
    expect(secondLeaseId).not.toBe(firstLeaseId);
    finishDisconnect();
  });

  it("validates listener frames and rejects stale generations", async () => {
    const appState = makeAppState();
    const subscriptions = {
      hasPod: vi.fn().mockReturnValue(true),
      sendToPod: vi.fn(),
    };
    const registry = { broadcast: vi.fn() };
    const wiring = new RelayListenerWiring(
      appState as unknown as AppState,
      subscriptions as unknown as RelayOutputSubscriptions,
      registry as unknown as WindowRegistry,
    );
    const lease = wiring.forPod("pod-1");
    expect(wiring.forPod("pod-1")).toBe(lease);

    lease.onBound(null, 2);
    lease.onBound(null, 0);
    lease.onBound(null, Number.NaN);
    lease.onStatus(null, "not-json");
    lease.onAcp(null, "not-json");
    lease.onStatus(null, '{"generation":2,"revision":1}');
    lease.onAcp(null, '{"generation":2,"msgType":13}');
    wiring.handleDriverDisconnected('{"podKey":"pod-1","generation":1}');

    expect(subscriptions.sendToPod).toHaveBeenCalledWith("pod-1", "relay:status", {
      podKey: "pod-1",
      json: '{"generation":2,"revision":1}',
    });
    expect(subscriptions.sendToPod).toHaveBeenCalledWith("pod-1", "relay:acp", {
      podKey: "pod-1",
      json: '{"generation":2,"msgType":13}',
    });
    expect(appState.relayBindPodListeners).not.toHaveBeenCalled();
    expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
      "debug",
      "relay",
      "ignore stale pod disconnect pod-1/1",
    );
    expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
      "warn",
      "relay",
      "malformed status event for pod-1",
    );
    expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
      "warn",
      "relay",
      "malformed ACP event for pod-1",
    );
  });

  it("advances the lease from status and ACP events published by newer generations", () => {
    const appState = makeAppState();
    const subscriptions = {
      hasPod: vi.fn().mockReturnValue(true),
      sendToPod: vi.fn(),
    };
    const wiring = new RelayListenerWiring(
      appState as unknown as AppState,
      subscriptions as unknown as RelayOutputSubscriptions,
      { broadcast: vi.fn() } as unknown as WindowRegistry,
    );
    const lease = wiring.forPod("pod-1");
    lease.onBound(null, 1);

    lease.onStatus(null, '{"generation":2,"revision":1,"status":"connected"}');
    lease.onAcp(null, '{"generation":3,"msgType":13,"payload":{"id":"new"}}');
    lease.onStatus(null, '{"generation":2,"revision":2,"status":"error"}');

    expect(subscriptions.sendToPod).toHaveBeenCalledWith("pod-1", "relay:status", {
      podKey: "pod-1",
      json: '{"generation":2,"revision":1,"status":"connected"}',
    });
    expect(subscriptions.sendToPod).toHaveBeenCalledWith("pod-1", "relay:acp", {
      podKey: "pod-1",
      json: '{"generation":3,"msgType":13,"payload":{"id":"new"}}',
    });
    expect(subscriptions.sendToPod).toHaveBeenCalledTimes(2);
  });

  it("does not drop a replacement lease installed while checking route ownership", () => {
    const appState = makeAppState();
    let replacement!: RelayPodListeners;
    const subscriptions = {
      hasPod: vi.fn(() => {
        wiring.dropPod("pod-1");
        replacement = wiring.forPod("pod-1");
        return false;
      }),
      sendToPod: vi.fn(),
    };
    const registry = { broadcast: vi.fn() };
    const wiring = new RelayListenerWiring(
      appState as unknown as AppState,
      subscriptions as unknown as RelayOutputSubscriptions,
      registry as unknown as WindowRegistry,
    );
    const retired = wiring.forPod("pod-1");
    retired.onBound(null, 1);

    wiring.handleDriverDisconnected('{"podKey":"pod-1","generation":1}');
    retired.onStatus(null, '{"generation":2,"revision":1}');

    expect(wiring.forPod("pod-1")).toBe(replacement);
    expect(subscriptions.sendToPod).not.toHaveBeenCalled();
    expect(registry.broadcast).toHaveBeenCalledExactlyOnceWith(
      "relay:pod-disconnected",
      { podKey: "pod-1", generation: 1 },
    );
  });

  it("coalesces listener rebinds and logs a failed rebind", async () => {
    const appState = makeAppState();
    let rejectRebind!: (error: Error) => void;
    appState.relayBindPodListeners.mockImplementationOnce(
      () => new Promise<number>((_resolve, reject) => { rejectRebind = reject; }),
    );
    const subscriptions = {
      hasPod: vi.fn().mockReturnValue(true),
      sendToPod: vi.fn(),
    };
    const wiring = new RelayListenerWiring(
      appState as unknown as AppState,
      subscriptions as unknown as RelayOutputSubscriptions,
      { broadcast: vi.fn() } as unknown as WindowRegistry,
    );
    const lease = wiring.forPod("pod-1");
    lease.onBound(null, 1);

    wiring.handleDriverDisconnected('{"podKey":"pod-1","generation":1}');
    wiring.handleDriverDisconnected('{"podKey":"pod-1","generation":1}');
    expect(appState.relayBindPodListeners).toHaveBeenCalledTimes(1);
    rejectRebind(new Error("bridge unavailable"));

    await vi.waitFor(() => expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
      "warn",
      "relay",
      "listener rebind pod-1 failed: Error: bridge unavailable",
    ));

    appState.relayBindPodListeners.mockResolvedValueOnce(2);
    wiring.handleDriverDisconnected('{"podKey":"pod-1","generation":1}');
    await vi.waitFor(() => expect(appState.relayBindPodListeners).toHaveBeenCalledTimes(2));
  });

  it("retires a final lease, broadcasts once, and ignores its late callbacks", () => {
    const appState = makeAppState();
    const subscriptions = {
      hasPod: vi.fn().mockReturnValue(false),
      sendToPod: vi.fn(),
    };
    const registry = { broadcast: vi.fn() };
    const wiring = new RelayListenerWiring(
      appState as unknown as AppState,
      subscriptions as unknown as RelayOutputSubscriptions,
      registry as unknown as WindowRegistry,
    );
    const lease = wiring.forPod("pod-1");
    lease.onBound(null, 1);

    wiring.handleDriverDisconnected("not-json");
    wiring.handleDriverDisconnected('{"podKey":"pod-1","generation":1}');
    lease.onStatus(null, '{"generation":2,"revision":1}');
    lease.onAcp(null, '{"generation":2,"msgType":13}');

    expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
      "warn",
      "relay",
      "malformed pod disconnect: not-json",
    );
    expect(registry.broadcast).toHaveBeenCalledExactlyOnceWith(
      "relay:pod-disconnected",
      { podKey: "pod-1", generation: 1 },
    );
    expect(subscriptions.sendToPod).not.toHaveBeenCalled();
  });

  it("drops unused leases and clears all remaining leases", () => {
    const appState = makeAppState();
    const subscriptions = {
      hasPod: vi.fn().mockReturnValueOnce(true).mockReturnValue(false),
      sendToPod: vi.fn(),
    };
    const wiring = new RelayListenerWiring(
      appState as unknown as AppState,
      subscriptions as unknown as RelayOutputSubscriptions,
      { broadcast: vi.fn() } as unknown as WindowRegistry,
    );
    const retained = wiring.forPod("retained");
    wiring.dropPodIfUnused("retained");
    expect(wiring.forPod("retained")).toBe(retained);

    const removed = wiring.forPod("removed");
    wiring.dropPodIfUnused("removed");
    expect(wiring.forPod("removed")).not.toBe(removed);

    const beforeClear = wiring.forPod("clear-me");
    wiring.clear();
    expect(wiring.forPod("clear-me")).not.toBe(beforeClear);
  });

  it("creates a lease during rebind and rejects every late callback after retirement", async () => {
    const appState = makeAppState();
    let resolveRebind!: (generation: number) => void;
    appState.relayBindPodListeners.mockImplementationOnce(
      () => new Promise<number>((resolve) => { resolveRebind = resolve; }),
    );
    const subscriptions = {
      hasPod: vi.fn().mockReturnValue(true),
      sendToPod: vi.fn(),
    };
    const wiring = new RelayListenerWiring(
      appState as unknown as AppState,
      subscriptions as unknown as RelayOutputSubscriptions,
      { broadcast: vi.fn() } as unknown as WindowRegistry,
    );

    wiring.handleDriverDisconnected('{"podKey":"pod-1","generation":1}');
    const lease = wiring.forPod("pod-1");
    lease.onBound(null, 3);
    lease.onStatus(null, '{"generation":2,"revision":9}');
    lease.onAcp(null, '{"generation":2,"msgType":13}');
    lease.onBound(null, 2);
    wiring.dropPod("pod-1");
    lease.onBound(null, 4);
    resolveRebind(4);

    await vi.waitFor(() => expect(appState.relayBindPodListeners).toHaveBeenCalledTimes(1));
    expect(subscriptions.sendToPod).not.toHaveBeenCalled();
  });
});
