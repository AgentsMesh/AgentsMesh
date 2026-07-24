import { beforeEach, describe, expect, it, vi } from "vitest";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";
import type { RelayOutputSubscriptions } from "./relay_output_subscriptions";
import type { WindowRegistry } from "./window_registry";

type IpcHandler = (event: { sender: { id: number } }, ...args: unknown[]) => unknown;

const ipc = vi.hoisted(() => ({
  handlers: new Map<string, IpcHandler>(),
  handle: vi.fn((channel: string, handler: IpcHandler) => {
    ipc.handlers.set(channel, handler);
  }),
  removeHandler: vi.fn(),
}));

vi.mock("electron", () => ({ ipcMain: ipc }));
vi.mock("@agentsmesh/node-bridge", () => ({ logEvent: vi.fn() }));

import { setupRelayBridge } from "./relay";
import {
  RelayListenerWiring,
  type RelayPodListeners,
} from "./relay_listener_wiring";
import {
  createRelayBridgeTestHarness,
  makeAppState,
} from "./relay_bridge.fixture.test";

const { invoke, reset: resetRelayBridgeFixture, subscribe } =
  createRelayBridgeTestHarness(ipc);

describe("relay listener bridge wiring", () => {
  beforeEach(resetRelayBridgeFixture);

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

});
