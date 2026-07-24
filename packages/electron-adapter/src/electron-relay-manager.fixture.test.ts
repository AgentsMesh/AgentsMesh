import { afterEach, beforeEach, vi } from "vitest";
import type {
  RelayAcpPayload,
  RelayOutputPayload,
  RelayPodDisconnectedPayload,
  RelayStatusPayload,
} from "./relay-ipc-contract";

export function installElectronRelayManagerFixture() {
  let emitOutputHandler!: (payload: RelayOutputPayload) => void;
  let emitStatusHandler!: (payload: RelayStatusPayload) => void;
  let emitAcpHandler!: (payload: RelayAcpPayload) => void;
  let emitDisconnectedHandler!: (payload: RelayPodDisconnectedPayload) => void;
  const invoke = vi.fn();

  beforeEach(() => {
    invoke.mockReset().mockImplementation((channel: string, ...args: unknown[]) => {
      if (channel === "relay:subscribe") {
        queueMicrotask(() => {
          emitOutputHandler({
            podKey: args[0] as string,
            subId: args[1] as string,
            attemptId: args[2] as string,
            data: new Uint8Array(),
          });
        });
      }
      return Promise.resolve(undefined);
    });
    const electronAPI = {
      invoke,
      onRelayOutput: (handler: (payload: RelayOutputPayload) => void) => {
        emitOutputHandler = handler;
        return vi.fn();
      },
      onRelayStatus: (handler: (payload: RelayStatusPayload) => void) => {
        emitStatusHandler = handler;
        return vi.fn();
      },
      onRelayAcp: (handler: (payload: RelayAcpPayload) => void) => {
        emitAcpHandler = handler;
        return vi.fn();
      },
      onRelayPodDisconnected: (handler: (payload: RelayPodDisconnectedPayload) => void) => {
        emitDisconnectedHandler = handler;
        return vi.fn();
      },
    };
    (globalThis as { window?: unknown }).window = { electronAPI };
  });

  afterEach(() => {
    delete (globalThis as { window?: unknown }).window;
  });

  return {
    invoke,
    emitStatus: (payload: RelayStatusPayload) => emitStatusHandler(payload),
    emitAcp: (payload: RelayAcpPayload) => emitAcpHandler(payload),
    emitDisconnected: (payload: RelayPodDisconnectedPayload) => emitDisconnectedHandler(payload),
    emitFromSubscribe: (call: number, data: Uint8Array) => {
      const args = invoke.mock.calls[call];
      emitOutputHandler({ podKey: args[1], subId: args[2], attemptId: args[3], data });
    },
  };
}
