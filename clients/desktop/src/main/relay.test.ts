import { beforeEach, describe, expect, it, vi } from "vitest";
import { logEvent, type AppState } from "@agentsmesh/node-bridge";
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

function event(wcId: number) {
  return { sender: { id: wcId } };
}

async function invoke(channel: string, wcId: number, ...args: unknown[]) {
  const handler = ipc.handlers.get(channel);
  if (!handler) throw new Error(`missing IPC handler ${channel}`);
  return handler(event(wcId), ...args);
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

const flushOutput = () => new Promise<void>((resolve) => setImmediate(resolve));

describe("relay main-process bridge", () => {
  beforeEach(() => {
    ipc.handlers.clear();
    vi.clearAllMocks();
    nextAttempt = 0;
  });

  it("keeps a distinct Rust subscriber and output route per renderer subscription", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    setupRelayBridge(
      appState as unknown as AppState,
      registry as unknown as WindowRegistry,
    );

    await subscribe(11, "pod-1", "terminal");
    await subscribe(22, "pod-1", "terminal");

    expect(appState.relaySubscribe).toHaveBeenCalledTimes(2);
    const [first, second] = appState.relaySubscribe.mock.calls;
    expect(first[1]).toContain("desktop:11:");
    expect(second[1]).toContain("desktop:22:");
    expect(first[1]).not.toBe(second[1]);
    expect(first[5]).toBeTypeOf("function");
    expect(first[6]).toBeTypeOf("function");

    const firstOutput = first[4] as (_error: unknown, bytes: number[]) => void;
    const secondOutput = second[4] as (_error: unknown, bytes: number[]) => void;
    firstOutput(null, [1, 2]);
    secondOutput(null, [3]);
    await flushOutput();

    expect(registry.sendTo).toHaveBeenCalledTimes(2);
    expect(registry.sendTo).toHaveBeenCalledWith(11, "relay:output", {
      podKey: "pod-1",
      subId: "terminal",
      attemptId: "attempt:1",
      data: Uint8Array.of(1, 2),
    });
    expect(registry.sendTo).toHaveBeenCalledWith(22, "relay:output", {
      podKey: "pod-1",
      subId: "terminal",
      attemptId: "attempt:2",
      data: Uint8Array.of(3),
    });
  });

  it("forwards an empty baseline marker through the output batch", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    setupRelayBridge(
      appState as unknown as AppState,
      registry as unknown as WindowRegistry,
    );
    await subscribe(11, "pod-1", "terminal");
    const output = appState.relaySubscribe.mock.calls[0][4] as
      (_error: unknown, bytes: number[]) => void;

    output(null, []);
    await flushOutput();

    expect(registry.sendTo).toHaveBeenCalledExactlyOnceWith(11, "relay:output", {
      podKey: "pod-1",
      subId: "terminal",
      attemptId: "attempt:1",
      data: new Uint8Array(),
    });
  });

  it("unsubscribes one window's core subscribers without disconnecting shared pods", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    setupRelayBridge(
      appState as unknown as AppState,
      registry as unknown as WindowRegistry,
    );
    await subscribe(11, "pod-1", "one");
    await subscribe(22, "pod-1", "two");
    const firstCoreId = appState.relaySubscribe.mock.calls[0][1];

    await invoke("relay:disconnect", 11, "pod-1");

    expect(appState.relayUnsubscribe).toHaveBeenCalledExactlyOnceWith("pod-1", firstCoreId);
    expect(appState.relayDisconnect).not.toHaveBeenCalled();

    await invoke("relay:disconnect", 22, "pod-1");
    expect(appState.relayDisconnect).toHaveBeenCalledExactlyOnceWith("pod-1");
  });

  it("drops queued output when its renderer unsubscribes before the IPC flush", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    setupRelayBridge(
      appState as unknown as AppState,
      registry as unknown as WindowRegistry,
    );
    await subscribe(11, "pod-1", "one");
    const output = appState.relaySubscribe.mock.calls[0][4] as
      (_error: unknown, bytes: number[]) => void;

    output(null, [9]);
    await invoke("relay:unsubscribe", 11, "pod-1", "one");
    await flushOutput();

    expect(registry.sendTo).not.toHaveBeenCalled();
  });

  it("releases both active and candidate subscribers when replacement is in flight", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    setupRelayBridge(
      appState as unknown as AppState,
      registry as unknown as WindowRegistry,
    );
    await subscribe(11, "pod-1", "one");
    const activeId = appState.relaySubscribe.mock.calls[0][1];
    let resolveReplacement!: () => void;
    appState.relaySubscribe.mockImplementationOnce(
      () => new Promise<void>((resolve) => { resolveReplacement = resolve; }),
    );

    const replacement = subscribe(11, "pod-1", "one");
    await vi.waitFor(() => expect(appState.relaySubscribe).toHaveBeenCalledTimes(2));
    const candidateId = appState.relaySubscribe.mock.calls[1][1];
    await invoke("relay:unsubscribe", 11, "pod-1", "one");

    expect(appState.relayUnsubscribe).toHaveBeenCalledWith("pod-1", activeId);
    expect(appState.relayUnsubscribe).toHaveBeenCalledWith("pod-1", candidateId);

    resolveReplacement();
    await replacement;
  });

  it("restores the previous route when a replacement subscribe fails", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    setupRelayBridge(
      appState as unknown as AppState,
      registry as unknown as WindowRegistry,
    );
    await subscribe(11, "pod-1", "one");
    const original = appState.relaySubscribe.mock.calls[0];
    const originalCoreId = original[1];
    const originalOutput = original[4] as (_error: unknown, bytes: number[]) => void;
    let rejectReplacement!: (error: Error) => void;
    appState.relaySubscribe.mockImplementationOnce(
      () => new Promise<void>((_resolve, reject) => { rejectReplacement = reject; }),
    );

    const replacement = subscribe(11, "pod-1", "one");
    const rejected = expect(replacement).rejects.toThrow("subscribe failed");

    originalOutput(null, [4, 2]);
    await flushOutput();
    expect(registry.sendTo).toHaveBeenCalledWith(11, "relay:output", {
      podKey: "pod-1",
      subId: "one",
      attemptId: "attempt:1",
      data: Uint8Array.of(4, 2),
    });

    rejectReplacement(new Error("subscribe failed"));
    await rejected;

    const failedCoreId = appState.relaySubscribe.mock.calls[1][1];
    expect(failedCoreId).not.toBe(originalCoreId);
    expect(appState.relayUnsubscribe).toHaveBeenCalledWith("pod-1", failedCoreId);
    expect(appState.relayUnsubscribe).not.toHaveBeenCalledWith("pod-1", originalCoreId);

    registry.sendTo.mockClear();
    originalOutput(null, [8, 4]);
    await flushOutput();
    expect(registry.sendTo).toHaveBeenCalledWith(11, "relay:output", {
      podKey: "pod-1",
      subId: "one",
      attemptId: "attempt:1",
      data: Uint8Array.of(8, 4),
    });
  });

  it("cancels a superseded candidate instead of leaking a pending core subscriber", async () => {
    const appState = makeAppState();
    const registry = { sendTo: vi.fn() };
    setupRelayBridge(
      appState as unknown as AppState,
      registry as unknown as WindowRegistry,
    );
    await subscribe(11, "pod-1", "one");
    let resolveOlder!: () => void;
    appState.relaySubscribe.mockImplementationOnce(
      () => new Promise<void>((resolve) => { resolveOlder = resolve; }),
    );

    const older = subscribe(11, "pod-1", "one");
    await vi.waitFor(() => expect(appState.relaySubscribe).toHaveBeenCalledTimes(2));
    const olderCoreId = appState.relaySubscribe.mock.calls[1][1];
    const newer = subscribe(11, "pod-1", "one");

    expect(appState.relayUnsubscribe).toHaveBeenCalledWith("pod-1", olderCoreId);
    await expect(newer).resolves.toBe(true);
    resolveOlder();
    await expect(older).resolves.toBe(false);
  });

  it("forwards every command and query IPC channel with its typed arguments", async () => {
    const appState = makeAppState();
    setupRelayBridge(
      appState as unknown as AppState,
      { sendTo: vi.fn() } as unknown as WindowRegistry,
    );

    await invoke("relay:send", 11, "pod-1", "input");
    await invoke("relay:resize", 11, "pod-1", 100, 40);
    await invoke("relay:forceResize", 11, "pod-1", 120, 50);
    await invoke("relay:acpCommand", 11, "pod-1", '{"type":"cancel"}');
    await expect(invoke("relay:getStatus", 11, "pod-1")).resolves.toBe("connected");
    await expect(invoke("relay:isRunnerDisconnected", 11, "pod-1")).resolves.toBe(false);
    await expect(invoke("relay:getPodSize", 11, "pod-1")).resolves.toEqual([80, 24]);
    await expect(invoke("relay:unsubscribe", 11, "missing", "missing")).resolves.toBeUndefined();
    await expect(invoke("relay:disconnect", 11, "missing")).resolves.toBeUndefined();
    await expect(invoke("relay:disconnectAll", 11)).resolves.toBeUndefined();

    expect(appState.relaySend).toHaveBeenCalledExactlyOnceWith("pod-1", "input");
    expect(appState.relaySendResize).toHaveBeenCalledExactlyOnceWith("pod-1", 100, 40);
    expect(appState.relayForceResize).toHaveBeenCalledExactlyOnceWith("pod-1", 120, 50);
    expect(appState.relaySendAcpCommand).toHaveBeenCalledExactlyOnceWith(
      "pod-1",
      '{"type":"cancel"}',
    );
  });

  it("releases all subscriptions owned by a destroyed webContents", async () => {
    const appState = makeAppState();
    const bridge = setupRelayBridge(
      appState as unknown as AppState,
      { sendTo: vi.fn() } as unknown as WindowRegistry,
    );
    await subscribe(11, "pod-1", "one");
    await subscribe(22, "pod-1", "shared");
    await subscribe(11, "pod-2", "two");
    const firstId = appState.relaySubscribe.mock.calls[0][1];
    const secondPodId = appState.relaySubscribe.mock.calls[2][1];
    appState.relayUnsubscribe.mockRejectedValue(new Error("core already closed"));

    bridge.releaseWebContents(11);

    await vi.waitFor(() => expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
      "warn",
      "relay",
      expect.stringContaining("relayUnsubscribe pod-1/"),
    ));
    expect(appState.relayUnsubscribe).toHaveBeenCalledWith("pod-1", firstId);
    expect(appState.relayUnsubscribe).toHaveBeenCalledWith("pod-2", secondPodId);
    expect(appState.relayDisconnect).not.toHaveBeenCalled();
  });

  it("logs cleanup failures for rollback, replacement, and superseded attempts", async () => {
    const appState = makeAppState();
    setupRelayBridge(
      appState as unknown as AppState,
      { sendTo: vi.fn() } as unknown as WindowRegistry,
    );

    appState.relaySubscribe.mockRejectedValueOnce(new Error("subscribe failed"));
    appState.relayUnsubscribe.mockRejectedValueOnce(new Error("rollback failed"));
    await expect(subscribe(11, "pod-rollback", "terminal")).rejects.toThrow("subscribe failed");
    await vi.waitFor(() => expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
      "warn",
      "relay",
      expect.stringContaining("rollback pod-rollback/"),
    ));

    await subscribe(11, "pod-1", "terminal");
    appState.relayUnsubscribe.mockRejectedValue(new Error("cleanup failed"));
    let resolveOlder!: () => void;
    appState.relaySubscribe.mockImplementationOnce(
      () => new Promise<void>((resolve) => { resolveOlder = resolve; }),
    );
    const older = subscribe(11, "pod-1", "terminal");
    await vi.waitFor(() => expect(appState.relaySubscribe).toHaveBeenCalledTimes(3));
    await expect(subscribe(11, "pod-1", "terminal")).resolves.toBe(true);
    resolveOlder();
    await expect(older).resolves.toBe(false);

    await vi.waitFor(() => {
      expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
        "warn",
        "relay",
        expect.stringContaining("supersede pod-1/"),
      );
      expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
        "warn",
        "relay",
        expect.stringContaining("superseded pod-1/"),
      );
      expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
        "warn",
        "relay",
        expect.stringContaining("replace pod-1/"),
      );
    });
  });

  it("disposes every IPC handler and reports Rust pool teardown failure", async () => {
    const appState = makeAppState();
    appState.relayDisconnectAll.mockRejectedValueOnce(new Error("teardown failed"));
    const bridge = setupRelayBridge(
      appState as unknown as AppState,
      { sendTo: vi.fn() } as unknown as WindowRegistry,
    );

    bridge.dispose();

    expect(ipc.removeHandler).toHaveBeenCalledTimes(11);
    await vi.waitFor(() => expect(vi.mocked(logEvent)).toHaveBeenCalledWith(
      "warn",
      "relay",
      "relayDisconnectAll failed: Error: teardown failed",
    ));
  });

  it("returns false when a superseded candidate fails after losing ownership", async () => {
    const appState = makeAppState();
    setupRelayBridge(
      appState as unknown as AppState,
      { sendTo: vi.fn() } as unknown as WindowRegistry,
    );
    await subscribe(11, "pod-1", "terminal");
    let rejectOlder!: (error: Error) => void;
    appState.relaySubscribe.mockImplementationOnce(
      () => new Promise<void>((_resolve, reject) => { rejectOlder = reject; }),
    );
    const older = subscribe(11, "pod-1", "terminal");
    await vi.waitFor(() => expect(appState.relaySubscribe).toHaveBeenCalledTimes(2));
    await expect(subscribe(11, "pod-1", "terminal")).resolves.toBe(true);

    rejectOlder(new Error("late failure"));

    await expect(older).resolves.toBe(false);
  });

  it("unsubscribes one shared route without dropping the pod listener lease", async () => {
    const appState = makeAppState();
    setupRelayBridge(
      appState as unknown as AppState,
      { sendTo: vi.fn() } as unknown as WindowRegistry,
    );
    await subscribe(11, "pod-1", "one");
    await subscribe(22, "pod-1", "two");
    const firstId = appState.relaySubscribe.mock.calls[0][1];

    await invoke("relay:unsubscribe", 11, "pod-1", "one");

    expect(appState.relayUnsubscribe).toHaveBeenCalledExactlyOnceWith("pod-1", firstId);
    expect(appState.relayDisconnect).not.toHaveBeenCalled();
  });

});
