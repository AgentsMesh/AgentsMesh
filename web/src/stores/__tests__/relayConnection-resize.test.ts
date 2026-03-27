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
    getPodConnection: vi.fn().mockResolvedValue({
      relay_url: "wss://relay.example.com",
      token: "test-token",
      pod_key: "pod-1",
    }),
  },
}));

describe("relayConnection - resize", () => {
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

  describe("sendResize", () => {
    it("should not throw for invalid dimensions", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      expect(() => pool.sendResize("pod-1", 0, 0)).not.toThrow();
      expect(() => pool.sendResize("pod-1", -1, 24)).not.toThrow();
    });

    it("should send resize message when connection is open", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const conn = pool.getConnection("pod-1");
      expect(conn).toBeDefined();
      expect(conn!.ws.readyState).toBe(MockWebSocket.OPEN);

      // sendResize is debounced, need to advance timer
      pool.sendResize("pod-1", 120, 40);
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
      pool.sendResize("unknown-pod", 80, 24);
      await vi.advanceTimersByTimeAsync(200);

      expect(pool.getConnection("unknown-pod")).toBeUndefined();
    });
  });

  describe("forceResize", () => {
    it("should send resize immediately when connection is open", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const conn = pool.getConnection("pod-1");
      expect(conn).toBeDefined();
      const sendCallsBefore = (conn!.ws.send as MockSend).mock.calls.length;

      pool.forceResize("pod-1", 100, 30);

      expect((conn!.ws.send as MockSend).mock.calls.length).toBe(sendCallsBefore + 1);
      const lastCall = (conn!.ws.send as MockSend).mock.calls[(conn!.ws.send as MockSend).mock.calls.length - 1];
      const sentData = lastCall[0] as Uint8Array;

      expect(sentData[0]).toBe(0x04); // MsgType.Resize
      expect((sentData[1] << 8) | sentData[2]).toBe(100); // cols
      expect((sentData[3] << 8) | sentData[4]).toBe(30);  // rows
    });

    it("should queue pendingResize when connection is connecting", async () => {
      const onMessage = vi.fn();

      const subscribePromise = pool.subscribe("pod-1", "sub-1", onMessage);
      await Promise.resolve();

      const conn = pool.getConnection("pod-1");
      expect(conn).toBeDefined();
      expect(conn!.ws.readyState).toBe(MockWebSocket.CONNECTING);

      pool.forceResize("pod-1", 80, 24);

      expect(conn!.pendingResize).toEqual({ rows: 24, cols: 80 });

      await vi.runAllTimersAsync();
      await subscribePromise;

      expect(conn!.pendingResize).toBeUndefined();

      const sendCalls = (conn!.ws.send as MockSend).mock.calls;
      const resizeCalls = sendCalls.filter((call: unknown[]) => {
        const data = call[0] as Uint8Array;
        return data[0] === 0x04; // MsgType.Resize
      });
      expect(resizeCalls.length).toBeGreaterThan(0);
    });

    it("should not throw for non-existent connection", () => {
      expect(() => pool.forceResize("unknown-pod", 80, 24)).not.toThrow();
    });

    it("should not send resize for invalid dimensions", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      const conn = pool.getConnection("pod-1");
      const sendCallsBefore = (conn!.ws.send as MockSend).mock.calls.length;

      pool.forceResize("pod-1", 0, 24);
      pool.forceResize("pod-1", 80, 0);
      pool.forceResize("pod-1", -1, 24);
      pool.forceResize("pod-1", 80, -1);

      expect((conn!.ws.send as MockSend).mock.calls.length).toBe(sendCallsBefore);
    });

    it("should send resize after reconnection", async () => {
      const onMessage1 = vi.fn();
      const onMessage2 = vi.fn();

      await pool.subscribe("pod-1", "sub-1", onMessage1);
      await vi.runAllTimersAsync();

      await pool.subscribe("pod-1", "sub-2", onMessage2);
      await vi.runAllTimersAsync();

      const conn = pool.getConnection("pod-1");
      expect(conn).toBeDefined();
      expect(conn!.ws.readyState).toBe(MockWebSocket.OPEN);

      const sendCallsBefore = (conn!.ws.send as MockSend).mock.calls.length;

      pool.forceResize("pod-1", 120, 40);

      expect((conn!.ws.send as MockSend).mock.calls.length).toBe(sendCallsBefore + 1);
      const lastCall = (conn!.ws.send as MockSend).mock.calls[(conn!.ws.send as MockSend).mock.calls.length - 1];
      const sentData = lastCall[0] as Uint8Array;
      expect(sentData[0]).toBe(0x04); // MsgType.Resize
    });
  });

  describe("getPodSize", () => {
    it("should return undefined for unknown pod", () => {
      expect(pool.getPodSize("unknown")).toBeUndefined();
    });
  });
});
