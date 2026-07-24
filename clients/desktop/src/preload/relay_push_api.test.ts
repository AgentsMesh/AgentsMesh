import { beforeEach, describe, expect, it, vi } from "vitest";
import type {
  RelayPushChannel,
  RelayPushPayload,
} from "@agentsmesh/electron-adapter/relay-ipc-contract";

const ipc = vi.hoisted(() => ({
  on: vi.fn(),
  removeListener: vi.fn(),
}));

vi.mock("electron", () => ({ ipcRenderer: ipc }));

import { relayPushApi } from "./relay_push_api";

type Registration = {
  channel: RelayPushChannel;
  register: (handler: (payload: never) => void) => () => void;
  payload: RelayPushPayload<RelayPushChannel>;
};

const registrations: Registration[] = [
  {
    channel: "relay:output",
    register: relayPushApi.onRelayOutput as Registration["register"],
    payload: {
      podKey: "pod-1",
      subId: "terminal",
      attemptId: "attempt-1",
      data: Uint8Array.of(1, 2),
    },
  },
  {
    channel: "relay:status",
    register: relayPushApi.onRelayStatus as Registration["register"],
    payload: { podKey: "pod-1", json: '{"status":"connected"}' },
  },
  {
    channel: "relay:acp",
    register: relayPushApi.onRelayAcp as Registration["register"],
    payload: { podKey: "pod-1", json: '{"msgType":13}' },
  },
  {
    channel: "relay:pod-disconnected",
    register: relayPushApi.onRelayPodDisconnected as Registration["register"],
    payload: { podKey: "pod-1", generation: 7 },
  },
];

describe("relayPushApi", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it.each(registrations)("forwards $channel and removes its exact listener", ({
    channel,
    register,
    payload,
  }) => {
    const handler = vi.fn();
    const dispose = register(handler);

    expect(ipc.on).toHaveBeenCalledExactlyOnceWith(channel, expect.any(Function));
    const listener = ipc.on.mock.calls[0][1] as (event: unknown, value: unknown) => void;
    listener({ sender: "main" }, payload);
    expect(handler).toHaveBeenCalledExactlyOnceWith(payload);

    dispose();
    expect(ipc.removeListener).toHaveBeenCalledExactlyOnceWith(channel, listener);
  });
});
