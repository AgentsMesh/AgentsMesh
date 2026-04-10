import { describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api/base";
import type { PoolContext } from "../relayConnectionWebSocket";
import { reconnectConnection } from "../relayConnectionWebSocket";
import type { RelayConnection, ConnectionHandle } from "../relayConnectionTypes";

function makeMockWs(readyState = 3): WebSocket {
  return { readyState, close: vi.fn(), onopen: null, onclose: null, onerror: null, onmessage: null, send: vi.fn() } as unknown as WebSocket;
}

function makeMockConn(podKey: string, overrides: Partial<RelayConnection> = {}): RelayConnection {
  return {
    ws: makeMockWs(3),
    podKey,
    status: "connected",
    lastActivity: Date.now(),
    subscribers: new Map([["sub-1", vi.fn()]]),
    reconnectAttempts: 0,
    reconnectTimer: null,
    disconnectTimer: null,
    snapshotTimer: null,
    snapshotReceived: false,
    relayUrl: "wss://relay.example.com",
    relayToken: "token",
    runnerDisconnected: false,
    ...overrides,
  };
}

function makeMockCtx(connections: Map<string, RelayConnection>, subscribeFn?: PoolContext["subscribe"]): PoolContext {
  return {
    connections,
    notifyStatusChange: vi.fn(),
    notifyAcpListeners: vi.fn(),
    createHandle: vi.fn((_pk, _sid) => ({ send: vi.fn(), unsubscribe: vi.fn() }) as ConnectionHandle),
    subscribe: subscribeFn ?? vi.fn(),
    maxReconnectAttempts: 50,
    baseReconnectDelay: 1000,
  };
}

describe("reconnectConnection", () => {
  it("stops on non-retryable 400 error and removes connection", async () => {
    const conn = makeMockConn("pod-1");
    const connections = new Map([["pod-1", conn]]);
    const ctx = makeMockCtx(connections, vi.fn().mockRejectedValue(new ApiError(400, "Bad Request")));

    await reconnectConnection(ctx, "pod-1");

    expect(connections.has("pod-1")).toBe(false);
    expect(ctx.notifyStatusChange).toHaveBeenCalledWith("pod-1");
  });

  it("stops on non-retryable 404 error", async () => {
    const conn = makeMockConn("pod-1");
    const connections = new Map([["pod-1", conn]]);
    const ctx = makeMockCtx(connections, vi.fn().mockRejectedValue(new ApiError(404, "Not Found")));

    await reconnectConnection(ctx, "pod-1");

    expect(connections.has("pod-1")).toBe(false);
  });

  it("creates stub and keeps connection on retryable 503 error", async () => {
    const conn = makeMockConn("pod-1");
    const connections = new Map([["pod-1", conn]]);
    const ctx = makeMockCtx(connections, vi.fn().mockRejectedValue(new ApiError(503, "Service Unavailable")));

    await reconnectConnection(ctx, "pod-1");

    expect(connections.has("pod-1")).toBe(true);
    const stub = connections.get("pod-1")!;
    expect(stub.status).toBe("error");
    expect(stub.subscribers.size).toBe(1);
    expect(ctx.notifyStatusChange).toHaveBeenCalledWith("pod-1");
  });

  it("creates stub on generic network error", async () => {
    const conn = makeMockConn("pod-1");
    const connections = new Map([["pod-1", conn]]);
    const ctx = makeMockCtx(connections, vi.fn().mockRejectedValue(new Error("network timeout")));

    await reconnectConnection(ctx, "pod-1");

    expect(connections.has("pod-1")).toBe(true);
    expect(connections.get("pod-1")!.status).toBe("error");
  });

  it("preserves subscribers across successful reconnect", async () => {
    const subscriber1 = vi.fn();
    const subscriber2 = vi.fn();
    const conn = makeMockConn("pod-1", {
      subscribers: new Map([["sub-1", subscriber1], ["sub-2", subscriber2]]),
    });
    const connections = new Map([["pod-1", conn]]);

    const newConn = makeMockConn("pod-1", { subscribers: new Map() });
    const ctx = makeMockCtx(connections, vi.fn().mockImplementation(async (_pk, sid, cb) => {
      newConn.subscribers.set(sid, cb);
      connections.set("pod-1", newConn);
      return { send: vi.fn(), unsubscribe: vi.fn() };
    }));

    await reconnectConnection(ctx, "pod-1");

    const result = connections.get("pod-1")!;
    expect(result.subscribers.size).toBe(2);
    expect(result.subscribers.has("sub-1")).toBe(true);
    expect(result.subscribers.has("sub-2")).toBe(true);
  });

  it("skips if no subscribers", async () => {
    const conn = makeMockConn("pod-1", { subscribers: new Map() });
    const connections = new Map([["pod-1", conn]]);
    const ctx = makeMockCtx(connections);

    await reconnectConnection(ctx, "pod-1");

    expect(ctx.subscribe).not.toHaveBeenCalled();
  });

  it("skips if connection not found", async () => {
    const connections = new Map<string, RelayConnection>();
    const ctx = makeMockCtx(connections);

    await reconnectConnection(ctx, "pod-1");

    expect(ctx.subscribe).not.toHaveBeenCalled();
  });
});
