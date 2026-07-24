import { vi } from "vitest";
import { registerServiceProvider, markServiceReady } from "@agentsmesh/service-runtime";
import { getPodConnection } from "@/lib/api/facade/podConnect";

// The adapter delegates connection management to the Rust-backed relay manager.
// Registering a real service provider is the supported test seam for this package.
export const mgr = {
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

export function lastStatusCb(): (raw: StatusRaw) => void {
  return mgr.on_status_change.mock.calls.at(-1)![1] as (raw: StatusRaw) => void;
}

export function lastAcpCb(): (messageType: number, payload: unknown) => void {
  return mgr.on_acp_message.mock.calls.at(-1)![1] as
    (messageType: number, payload: unknown) => void;
}

export function podDisconnectedCb(): (podKey: string) => void {
  return mgr.on_pod_disconnected.mock.calls.at(-1)![0] as (podKey: string) => void;
}

export function getPodConnectionMock() {
  return vi.mocked(getPodConnection);
}

export async function resetRelayConnectionPool() {
  vi.clearAllMocks();
  mgr.subscribe.mockResolvedValue(undefined);
  vi.mocked(getPodConnection).mockResolvedValue({
    relay_url: "wss://relay.example.com",
    token: "test-token",
    pod_key: "pod-1",
  } as never);
  registerServiceProvider({ relayManager: mgr as never });
  markServiceReady();
  return freshPool();
}
