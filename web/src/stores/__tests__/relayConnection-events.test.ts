import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  url: string;
  readyState: number = MockWebSocket.CONNECTING;
  binaryType: string = "blob";
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: ((e: unknown) => void) | null = null;
  onmessage: ((e: { data: unknown }) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN;
      this.onopen?.();
    }, 0);
  }

  send = vi.fn();
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.();
  });
}

global.WebSocket = MockWebSocket as unknown as typeof WebSocket;

// Mock pod API
vi.mock("@/lib/api/pod", () => ({
  podApi: {
    getPodConnection: vi.fn().mockResolvedValue({
      relay_url: "wss://relay.example.com",
      token: "test-token",
      pod_key: "pod-1",
    }),
  },
}));

describe("relayConnection - events", () => {
  let pool: typeof import("@/stores/relayConnection").relayPool;

  beforeEach(async () => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    vi.resetModules();
    const importedModule = await import("@/stores/relayConnection");
    pool = importedModule.relayPool;
  });

  afterEach(() => {
    pool?.disconnectAll();
    vi.useRealTimers();
  });

  describe("message handling", () => {
    it("should forward output message to subscriber", async () => {
      const onMessage = vi.fn();

      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const conn = pool.getConnection("pod-1");
      expect(conn).toBeDefined();

      // Create a mock output message (type 0x02 = Output)
      const payload = new TextEncoder().encode("Hello, World!");
      const message = new Uint8Array(1 + payload.length);
      message[0] = 0x02; // MsgType.Output
      message.set(payload, 1);

      conn!.ws.onmessage?.({ data: message.buffer } as MessageEvent);

      expect(onMessage).toHaveBeenCalledTimes(1);

      const received = onMessage.mock.calls[0][0] as Uint8Array;
      expect(Array.from(received)).toEqual(Array.from(payload));
    });
  });

  describe("onStatusChange", () => {
    it("should call listener immediately with current status (none for unknown pod)", () => {
      const listener = vi.fn();
      pool.onStatusChange("unknown-pod", listener);

      expect(listener).toHaveBeenCalledTimes(1);
      expect(listener).toHaveBeenCalledWith({
        status: "none",
        runnerDisconnected: false,
      });
    });

    it("should call listener immediately with current connected status", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener = vi.fn();
      pool.onStatusChange("pod-1", listener);

      expect(listener).toHaveBeenCalledTimes(1);
      expect(listener).toHaveBeenCalledWith({
        status: "connected",
        runnerDisconnected: false,
      });
    });

    it("should notify listener when connection status changes to connected", async () => {
      const listener = vi.fn();
      pool.onStatusChange("pod-1", listener);

      expect(listener).toHaveBeenCalledWith({
        status: "none",
        runnerDisconnected: false,
      });

      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const calls = listener.mock.calls.map((c) => c[0].status);
      expect(calls).toContain("connecting");
      expect(calls).toContain("connected");
    });

    it("should notify listener when disconnected", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener = vi.fn();
      pool.onStatusChange("pod-1", listener);
      listener.mockClear();

      pool.disconnect("pod-1");

      expect(listener).toHaveBeenCalledWith({
        status: "none",
        runnerDisconnected: false,
      });
    });

    it("should notify listener when runner disconnects", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener = vi.fn();
      pool.onStatusChange("pod-1", listener);
      listener.mockClear();

      const conn = pool.getConnection("pod-1");
      expect(conn).toBeDefined();
      const message = new Uint8Array([0x08]); // MsgType.RunnerDisconnected
      conn!.ws.onmessage?.({ data: message.buffer } as MessageEvent);

      expect(listener).toHaveBeenCalledWith({
        status: "connected",
        runnerDisconnected: true,
      });
    });

    it("should notify listener when runner reconnects", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const conn = pool.getConnection("pod-1");
      expect(conn).toBeDefined();

      const disconnectMsg = new Uint8Array([0x08]);
      conn!.ws.onmessage?.({ data: disconnectMsg.buffer } as MessageEvent);

      const listener = vi.fn();
      pool.onStatusChange("pod-1", listener);

      expect(listener).toHaveBeenCalledWith({
        status: "connected",
        runnerDisconnected: true,
      });
      listener.mockClear();

      const reconnectMsg = new Uint8Array([0x09]); // MsgType.RunnerReconnected
      conn!.ws.onmessage?.({ data: reconnectMsg.buffer } as MessageEvent);

      expect(listener).toHaveBeenCalledWith({
        status: "connected",
        runnerDisconnected: false,
      });
    });

    it("should support multiple listeners for same pod", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener1 = vi.fn();
      const listener2 = vi.fn();
      pool.onStatusChange("pod-1", listener1);
      pool.onStatusChange("pod-1", listener2);
      listener1.mockClear();
      listener2.mockClear();

      pool.disconnect("pod-1");

      expect(listener1).toHaveBeenCalled();
      expect(listener2).toHaveBeenCalled();
    });

    it("should stop notifying after unsubscribe", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener = vi.fn();
      const unsubscribe = pool.onStatusChange("pod-1", listener);
      listener.mockClear();

      unsubscribe();

      pool.disconnect("pod-1");

      expect(listener).not.toHaveBeenCalled();
    });

    it("should clean up listener set when last listener unsubscribes", () => {
      const listener = vi.fn();
      const unsubscribe = pool.onStatusChange("pod-1", listener);

      unsubscribe();

      const listener2 = vi.fn();
      pool.onStatusChange("pod-1", listener2);
      expect(listener2).toHaveBeenCalledTimes(1);
    });

    it("should notify on WebSocket error", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener = vi.fn();
      pool.onStatusChange("pod-1", listener);
      listener.mockClear();

      const conn = pool.getConnection("pod-1");
      conn!.ws.onerror?.(new Event("error"));

      expect(listener).toHaveBeenCalledWith({
        status: "error",
        runnerDisconnected: false,
      });
    });
  });
});
