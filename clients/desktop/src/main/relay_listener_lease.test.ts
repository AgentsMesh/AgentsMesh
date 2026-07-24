import { beforeEach, describe, expect, it, vi } from "vitest";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";
import type { RelayOutputSubscriptions } from "./relay_output_subscriptions";
import type { WindowRegistry } from "./window_registry";

vi.mock("@agentsmesh/node-bridge", () => ({ logEvent: vi.fn() }));

import {
  RelayListenerWiring,
  type RelayPodListeners,
} from "./relay_listener_wiring";
import { makeAppState } from "./relay_bridge.fixture.test";

describe("relay listener lease generations", () => {
  beforeEach(() => {
    vi.clearAllMocks();
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
