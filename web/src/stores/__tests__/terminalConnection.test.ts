import { describe, expect, it, vi, beforeEach, afterEach, type Mock } from "vitest";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type MockSend = Mock<(...args: any[]) => any>;

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
    getTerminalConnection: vi.fn().mockResolvedValue({
      relay_url: "wss://relay.example.com",
      token: "test-token",
      pod_key: "pod-1",
    }),
  },
}));

describe("terminalConnection", () => {
  let terminalPool: typeof import("@/stores/terminalConnection").terminalPool;

  beforeEach(async () => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    // Re-import to get fresh singleton
    vi.resetModules();
    const importedModule = await import("@/stores/terminalConnection");
    terminalPool = importedModule.terminalPool;
  });

  afterEach(() => {
    terminalPool?.disconnectAll();
    vi.useRealTimers();
  });

  describe("subscribe", () => {
    it("should create connection and return handle", async () => {
      const onMessage = vi.fn();
      const handlePromise = terminalPool.subscribe("pod-1", "sub-1", onMessage);

      await vi.runAllTimersAsync();
      const handle = await handlePromise;

      expect(handle).toHaveProperty("send");
      expect(handle).toHaveProperty("unsubscribe");
      expect(handle).toHaveProperty("disconnect");
      expect(terminalPool.getStatus("pod-1")).toBe("connected");
    });

    it("should add new subscriber to existing connection without reconnecting", async () => {
      const onMessage1 = vi.fn();
      const onMessage2 = vi.fn();

      await terminalPool.subscribe("pod-1", "sub-1", onMessage1);
      await vi.runAllTimersAsync();

      // New subscriber joins existing connection (no disconnect/reconnect)
      const handle2 = await terminalPool.subscribe("pod-1", "sub-2", onMessage2);
      await vi.runAllTimersAsync();

      expect(handle2).toHaveProperty("send");
      // Both subscribers should be registered on the same connection
      expect(terminalPool.getConnection("pod-1")?.subscribers.size).toBe(2);
      expect(terminalPool.getConnection("pod-1")?.subscribers.has("sub-1")).toBe(true);
      expect(terminalPool.getConnection("pod-1")?.subscribers.has("sub-2")).toBe(true);
    });

    it("should be idempotent - same subscriptionId replaces previous callback", async () => {
      const onMessage1 = vi.fn();
      const onMessage2 = vi.fn();

      await terminalPool.subscribe("pod-1", "sub-1", onMessage1);
      await vi.runAllTimersAsync();

      // Subscribe again with same subscriptionId
      await terminalPool.subscribe("pod-1", "sub-1", onMessage2);
      await vi.runAllTimersAsync();

      // Should still have only 1 subscriber (replaced, not added)
      expect(terminalPool.getConnection("pod-1")?.subscribers.size).toBe(1);
    });
  });

  describe("connect (deprecated)", () => {
    it("should create connection and return handle", async () => {
      const onMessage = vi.fn();
      const handlePromise = terminalPool.connect("pod-1", onMessage);

      await vi.runAllTimersAsync();
      const handle = await handlePromise;

      expect(handle).toHaveProperty("send");
      expect(handle).toHaveProperty("disconnect");
      expect(terminalPool.getStatus("pod-1")).toBe("connected");
    });

    it("should add new subscriber when called again for same pod", async () => {
      const onMessage1 = vi.fn();
      const onMessage2 = vi.fn();

      await terminalPool.connect("pod-1", onMessage1);
      await vi.runAllTimersAsync();

      // Second connect adds a new subscriber (legacy IDs are unique)
      const handle2 = await terminalPool.connect("pod-1", onMessage2);
      await vi.runAllTimersAsync();

      expect(handle2).toHaveProperty("send");
      // Both subscribers on same connection
      expect(terminalPool.getConnection("pod-1")?.subscribers.size).toBe(2);
    });
  });

  describe("unsubscribe", () => {
    it("should remove subscriber by subscriptionId", async () => {
      const onMessage = vi.fn();
      const handle = await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      handle.unsubscribe();

      // Subscriber should be removed
      expect(terminalPool.getConnection("pod-1")?.subscribers.size).toBe(0);
    });

    it("should delay disconnect when last subscriber leaves", async () => {
      const onMessage = vi.fn();
      const handle = await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      handle.unsubscribe();

      // Connection should still exist (delayed disconnect)
      expect(terminalPool.getConnection("pod-1")).toBeDefined();

      // Advance past disconnect delay (30s)
      await vi.advanceTimersByTimeAsync(30000);

      // Now connection should be gone
      expect(terminalPool.getConnection("pod-1")).toBeUndefined();
      expect(terminalPool.getStatus("pod-1")).toBe("none");
    });

    it("should cancel disconnect timer if new subscriber joins", async () => {
      const onMessage1 = vi.fn();
      const onMessage2 = vi.fn();

      const handle1 = await terminalPool.subscribe("pod-1", "sub-1", onMessage1);
      await vi.runAllTimersAsync();

      // Unsubscribe first subscriber
      handle1.unsubscribe();

      // Advance time partially (10s of 30s delay)
      await vi.advanceTimersByTimeAsync(10000);

      // New subscriber joins
      await terminalPool.subscribe("pod-1", "sub-2", onMessage2);

      // Advance past original disconnect time
      await vi.advanceTimersByTimeAsync(25000);

      // Connection should still exist (timer was cancelled)
      expect(terminalPool.getConnection("pod-1")).toBeDefined();
      expect(terminalPool.getConnection("pod-1")?.subscribers.size).toBe(1);
    });

    // Note: In the new architecture, multiple subscribers don't coexist on the same connection.
    // When a new subscriber joins, the connection is closed and recreated to get buffered output from Relay.
    // So "should not disconnect if other subscribers remain" scenario is no longer applicable.
  });

  describe("getStatus", () => {
    it("should return 'none' for unknown pod", () => {
      expect(terminalPool.getStatus("unknown")).toBe("none");
    });
  });

  describe("isConnected", () => {
    it("should return false for unknown pod", () => {
      expect(terminalPool.isConnected("unknown")).toBe(false);
    });
  });

  describe("isRunnerDisconnected", () => {
    it("should return false for unknown pod", () => {
      expect(terminalPool.isRunnerDisconnected("unknown")).toBe(false);
    });
  });

  describe("disconnect", () => {
    it("should close connection and remove from pool", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      terminalPool.disconnect("pod-1");

      expect(terminalPool.getStatus("pod-1")).toBe("none");
    });

    it("should be safe to call for non-existent pod", () => {
      expect(() => terminalPool.disconnect("unknown")).not.toThrow();
    });
  });

  describe("disconnectAll", () => {
    it("should disconnect all connections", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await terminalPool.subscribe("pod-2", "sub-2", onMessage);
      await vi.runAllTimersAsync();

      terminalPool.disconnectAll();

      expect(terminalPool.getStatus("pod-1")).toBe("none");
      expect(terminalPool.getStatus("pod-2")).toBe("none");
    });
  });

  describe("sendResize", () => {
    it("should not throw for invalid dimensions", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      expect(() => terminalPool.sendResize("pod-1", 0, 0)).not.toThrow();
      expect(() => terminalPool.sendResize("pod-1", -1, 24)).not.toThrow();
    });

    it("should send resize message when connection is open", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const conn = terminalPool.getConnection("pod-1");
      expect(conn).toBeDefined();
      expect(conn!.ws.readyState).toBe(MockWebSocket.OPEN);

      // sendResize is debounced, need to advance timer
      terminalPool.sendResize("pod-1", 120, 40);
      await vi.advanceTimersByTimeAsync(200); // debounce is 150ms

      // Verify resize message was sent
      expect(conn!.ws.send).toHaveBeenCalled();
      const lastCall = (conn!.ws.send as MockSend).mock.calls[(conn!.ws.send as MockSend).mock.calls.length - 1];
      const sentData = lastCall[0] as Uint8Array;

      // Message format: [MsgType.Resize(0x04), cols_hi, cols_lo, rows_hi, rows_lo]
      expect(sentData[0]).toBe(0x04); // MsgType.Resize
      expect((sentData[1] << 8) | sentData[2]).toBe(120); // cols
      expect((sentData[3] << 8) | sentData[4]).toBe(40);  // rows
    });

    it("should not send resize for non-existent connection", async () => {
      // No connection exists for "unknown-pod"
      terminalPool.sendResize("unknown-pod", 80, 24);
      await vi.advanceTimersByTimeAsync(200);

      // Should not throw, just silently do nothing
      expect(terminalPool.getConnection("unknown-pod")).toBeUndefined();
    });
  });

  describe("forceResize", () => {
    it("should send resize immediately when connection is open", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const conn = terminalPool.getConnection("pod-1");
      expect(conn).toBeDefined();
      const sendCallsBefore = (conn!.ws.send as MockSend).mock.calls.length;

      // forceResize should send immediately (no debounce)
      terminalPool.forceResize("pod-1", 100, 30);

      // Verify resize message was sent immediately
      expect((conn!.ws.send as MockSend).mock.calls.length).toBe(sendCallsBefore + 1);
      const lastCall = (conn!.ws.send as MockSend).mock.calls[(conn!.ws.send as MockSend).mock.calls.length - 1];
      const sentData = lastCall[0] as Uint8Array;

      // Message format: [MsgType.Resize(0x04), cols_hi, cols_lo, rows_hi, rows_lo]
      expect(sentData[0]).toBe(0x04); // MsgType.Resize
      expect((sentData[1] << 8) | sentData[2]).toBe(100); // cols
      expect((sentData[3] << 8) | sentData[4]).toBe(30);  // rows
    });

    it("should queue pendingResize when connection is connecting", async () => {
      const onMessage = vi.fn();

      // Start subscribe - this creates connection synchronously, but WebSocket opens async
      const subscribePromise = terminalPool.subscribe("pod-1", "sub-1", onMessage);

      // Need to wait for the promise to start (microtask), but not for WebSocket to open
      await Promise.resolve();

      // At this point, connection exists but WebSocket is still CONNECTING
      const conn = terminalPool.getConnection("pod-1");
      expect(conn).toBeDefined();
      expect(conn!.ws.readyState).toBe(MockWebSocket.CONNECTING);

      // forceResize while connecting should queue pendingResize
      terminalPool.forceResize("pod-1", 80, 24);

      // Verify pendingResize was set
      expect(conn!.pendingResize).toEqual({ rows: 24, cols: 80 });

      // Now let connection open
      await vi.runAllTimersAsync();
      await subscribePromise;

      // After connection opens, pendingResize should be sent and cleared
      expect(conn!.pendingResize).toBeUndefined();

      // Verify resize was actually sent
      const sendCalls = (conn!.ws.send as MockSend).mock.calls;
      const resizeCalls = sendCalls.filter((call: unknown[]) => {
        const data = call[0] as Uint8Array;
        return data[0] === 0x04; // MsgType.Resize
      });
      expect(resizeCalls.length).toBeGreaterThan(0);
    });

    it("should not throw for non-existent connection", () => {
      // No connection exists for "unknown-pod"
      expect(() => terminalPool.forceResize("unknown-pod", 80, 24)).not.toThrow();
    });

    it("should not send resize for invalid dimensions", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const conn = terminalPool.getConnection("pod-1");
      const sendCallsBefore = (conn!.ws.send as MockSend).mock.calls.length;

      // Invalid dimensions should be ignored
      terminalPool.forceResize("pod-1", 0, 24);
      terminalPool.forceResize("pod-1", 80, 0);
      terminalPool.forceResize("pod-1", -1, 24);
      terminalPool.forceResize("pod-1", 80, -1);

      // No resize messages should be sent
      expect((conn!.ws.send as MockSend).mock.calls.length).toBe(sendCallsBefore);
    });

    it("should send resize after reconnection", async () => {
      const onMessage1 = vi.fn();
      const onMessage2 = vi.fn();

      // First subscriber connects
      await terminalPool.subscribe("pod-1", "sub-1", onMessage1);
      await vi.runAllTimersAsync();

      // New subscriber causes reconnection
      await terminalPool.subscribe("pod-1", "sub-2", onMessage2);
      await vi.runAllTimersAsync();

      // Get the new connection
      const conn = terminalPool.getConnection("pod-1");
      expect(conn).toBeDefined();
      expect(conn!.ws.readyState).toBe(MockWebSocket.OPEN);

      const sendCallsBefore = (conn!.ws.send as MockSend).mock.calls.length;

      // forceResize on new connection should work
      terminalPool.forceResize("pod-1", 120, 40);

      // Verify resize was sent
      expect((conn!.ws.send as MockSend).mock.calls.length).toBe(sendCallsBefore + 1);
      const lastCall = (conn!.ws.send as MockSend).mock.calls[(conn!.ws.send as MockSend).mock.calls.length - 1];
      const sentData = lastCall[0] as Uint8Array;
      expect(sentData[0]).toBe(0x04); // MsgType.Resize
    });
  });

  describe("getPtySize", () => {
    it("should return undefined for unknown pod", () => {
      expect(terminalPool.getPtySize("unknown")).toBeUndefined();
    });
  });

  describe("message handling", () => {
    it("should forward output message to subscriber", async () => {
      const onMessage = vi.fn();

      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      // Simulate receiving a message
      const conn = terminalPool.getConnection("pod-1");
      expect(conn).toBeDefined();

      // Create a mock output message (type 0x02 = Output)
      const payload = new TextEncoder().encode("Hello, World!");
      const message = new Uint8Array(1 + payload.length);
      message[0] = 0x02; // MsgType.Output
      message.set(payload, 1);

      // Trigger onmessage
      conn!.ws.onmessage?.({ data: message.buffer } as MessageEvent);

      // Subscriber should be called
      expect(onMessage).toHaveBeenCalledTimes(1);

      // Verify it received the correct payload (Uint8Array comparison)
      const received = onMessage.mock.calls[0][0] as Uint8Array;
      expect(Array.from(received)).toEqual(Array.from(payload));
    });

    // Note: In the new architecture, multiple subscribers don't coexist on the same connection.
    // When a new subscriber joins, the connection is closed and recreated to get buffered output from Relay.
    // So "broadcast to all subscribers" test is no longer applicable - there's only ever 1 subscriber per connection.
  });

  describe("onStatusChange", () => {
    it("should call listener immediately with current status (none for unknown pod)", () => {
      const listener = vi.fn();
      terminalPool.onStatusChange("unknown-pod", listener);

      expect(listener).toHaveBeenCalledTimes(1);
      expect(listener).toHaveBeenCalledWith({
        status: "none",
        runnerDisconnected: false,
      });
    });

    it("should call listener immediately with current connected status", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener = vi.fn();
      terminalPool.onStatusChange("pod-1", listener);

      expect(listener).toHaveBeenCalledTimes(1);
      expect(listener).toHaveBeenCalledWith({
        status: "connected",
        runnerDisconnected: false,
      });
    });

    it("should notify listener when connection status changes to connected", async () => {
      const listener = vi.fn();
      terminalPool.onStatusChange("pod-1", listener);

      // Initial call with "none"
      expect(listener).toHaveBeenCalledWith({
        status: "none",
        runnerDisconnected: false,
      });

      // Subscribe triggers "connecting" then "connected"
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      // Should have been called with "connecting" and "connected"
      const calls = listener.mock.calls.map((c) => c[0].status);
      expect(calls).toContain("connecting");
      expect(calls).toContain("connected");
    });

    it("should notify listener when disconnected", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener = vi.fn();
      terminalPool.onStatusChange("pod-1", listener);
      listener.mockClear(); // Clear the initial call

      // Disconnect
      terminalPool.disconnect("pod-1");

      // Should be notified with "none" (connection removed from map)
      expect(listener).toHaveBeenCalledWith({
        status: "none",
        runnerDisconnected: false,
      });
    });

    it("should notify listener when runner disconnects", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener = vi.fn();
      terminalPool.onStatusChange("pod-1", listener);
      listener.mockClear();

      // Simulate RunnerDisconnected message (type 0x08)
      const conn = terminalPool.getConnection("pod-1");
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
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const conn = terminalPool.getConnection("pod-1");
      expect(conn).toBeDefined();

      // First disconnect runner
      const disconnectMsg = new Uint8Array([0x08]);
      conn!.ws.onmessage?.({ data: disconnectMsg.buffer } as MessageEvent);

      const listener = vi.fn();
      terminalPool.onStatusChange("pod-1", listener);

      // Initial call should show runner disconnected
      expect(listener).toHaveBeenCalledWith({
        status: "connected",
        runnerDisconnected: true,
      });
      listener.mockClear();

      // Now reconnect runner
      const reconnectMsg = new Uint8Array([0x09]); // MsgType.RunnerReconnected
      conn!.ws.onmessage?.({ data: reconnectMsg.buffer } as MessageEvent);

      expect(listener).toHaveBeenCalledWith({
        status: "connected",
        runnerDisconnected: false,
      });
    });

    it("should support multiple listeners for same pod", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener1 = vi.fn();
      const listener2 = vi.fn();
      terminalPool.onStatusChange("pod-1", listener1);
      terminalPool.onStatusChange("pod-1", listener2);
      listener1.mockClear();
      listener2.mockClear();

      // Trigger status change
      terminalPool.disconnect("pod-1");

      expect(listener1).toHaveBeenCalled();
      expect(listener2).toHaveBeenCalled();
    });

    it("should stop notifying after unsubscribe", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener = vi.fn();
      const unsubscribe = terminalPool.onStatusChange("pod-1", listener);
      listener.mockClear();

      // Unsubscribe
      unsubscribe();

      // Trigger status change
      terminalPool.disconnect("pod-1");

      // Should not have been called after unsubscribe
      expect(listener).not.toHaveBeenCalled();
    });

    it("should clean up listener set when last listener unsubscribes", () => {
      const listener = vi.fn();
      const unsubscribe = terminalPool.onStatusChange("pod-1", listener);

      unsubscribe();

      // Subscribe another listener and check it works fresh
      const listener2 = vi.fn();
      terminalPool.onStatusChange("pod-1", listener2);
      expect(listener2).toHaveBeenCalledTimes(1);
    });

    it("should notify on WebSocket error", async () => {
      const onMessage = vi.fn();
      await terminalPool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const listener = vi.fn();
      terminalPool.onStatusChange("pod-1", listener);
      listener.mockClear();

      // Simulate WebSocket error
      const conn = terminalPool.getConnection("pod-1");
      conn!.ws.onerror?.(new Event("error"));

      expect(listener).toHaveBeenCalledWith({
        status: "error",
        runnerDisconnected: false,
      });
    });
  });
});
