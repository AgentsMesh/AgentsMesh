import { vi } from "vitest";

type IpcHandler = (event: { sender: { id: number } }, ...args: unknown[]) => unknown;

type RelayIpcFixture = {
  handlers: Map<string, IpcHandler>;
};

export function makeAppState() {
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

export function createRelayBridgeTestHarness(ipc: RelayIpcFixture) {
  let nextAttempt = 0;

  const invoke = async (channel: string, wcId: number, ...args: unknown[]) => {
    const handler = ipc.handlers.get(channel);
    if (!handler) throw new Error(`missing IPC handler ${channel}`);
    return handler({ sender: { id: wcId } }, ...args);
  };

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

  return {
    invoke,
    subscribe,
    reset() {
      ipc.handlers.clear();
      vi.clearAllMocks();
      nextAttempt = 0;
    },
  };
}

export const flushOutput = () => new Promise<void>((resolve) => setImmediate(resolve));
