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

describe("relayConnection", () => {
  let pool: typeof import("@/stores/relayConnection").relayPool;

  beforeEach(async () => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    // Re-import to get fresh singleton
    vi.resetModules();
    const importedModule = await import("@/stores/relayConnection");
    pool = importedModule.relayPool;
  });

  afterEach(() => {
    pool?.disconnectAll();
    vi.useRealTimers();
  });

  describe("subscribe", () => {
    it("should create connection and return handle", async () => {
      const onMessage = vi.fn();
      const handlePromise = pool.subscribe("pod-1", "sub-1", onMessage);

      await vi.runAllTimersAsync();
      const handle = await handlePromise;

      expect(handle).toHaveProperty("send");
      expect(handle).toHaveProperty("unsubscribe");
      expect(pool.getStatus("pod-1")).toBe("connected");
    });

    it("should add new subscriber to existing connection without reconnecting", async () => {
      const onMessage1 = vi.fn();
      const onMessage2 = vi.fn();

      await pool.subscribe("pod-1", "sub-1", onMessage1);
      await vi.runAllTimersAsync();

      // New subscriber joins existing connection (no disconnect/reconnect)
      const handle2 = await pool.subscribe("pod-1", "sub-2", onMessage2);
      await vi.runAllTimersAsync();

      expect(handle2).toHaveProperty("send");
      // Both subscribers should be registered on the same connection
      expect(pool.getConnection("pod-1")?.subscribers.size).toBe(2);
      expect(pool.getConnection("pod-1")?.subscribers.has("sub-1")).toBe(true);
      expect(pool.getConnection("pod-1")?.subscribers.has("sub-2")).toBe(true);
    });

    it("should be idempotent - same subscriptionId replaces previous callback", async () => {
      const onMessage1 = vi.fn();
      const onMessage2 = vi.fn();

      await pool.subscribe("pod-1", "sub-1", onMessage1);
      await vi.runAllTimersAsync();

      // Subscribe again with same subscriptionId
      await pool.subscribe("pod-1", "sub-1", onMessage2);
      await vi.runAllTimersAsync();

      // Should still have only 1 subscriber (replaced, not added)
      expect(pool.getConnection("pod-1")?.subscribers.size).toBe(1);
    });
  });

  describe("unsubscribe", () => {
    it("should remove subscriber by subscriptionId", async () => {
      const onMessage = vi.fn();
      const handle = await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      handle.unsubscribe();

      // Subscriber should be removed
      expect(pool.getConnection("pod-1")?.subscribers.size).toBe(0);
    });

    it("should delay disconnect when last subscriber leaves", async () => {
      const onMessage = vi.fn();
      const handle = await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      handle.unsubscribe();

      // Connection should still exist (delayed disconnect)
      expect(pool.getConnection("pod-1")).toBeDefined();

      // Advance past disconnect delay (30s)
      await vi.advanceTimersByTimeAsync(30000);

      // Now connection should be gone
      expect(pool.getConnection("pod-1")).toBeUndefined();
      expect(pool.getStatus("pod-1")).toBe("none");
    });

    it("should cancel disconnect timer if new subscriber joins", async () => {
      const onMessage1 = vi.fn();
      const onMessage2 = vi.fn();

      const handle1 = await pool.subscribe("pod-1", "sub-1", onMessage1);
      await vi.runAllTimersAsync();

      // Unsubscribe first subscriber
      handle1.unsubscribe();

      // Advance time partially (10s of 30s delay)
      await vi.advanceTimersByTimeAsync(10000);

      // New subscriber joins
      await pool.subscribe("pod-1", "sub-2", onMessage2);

      // Advance past original disconnect time
      await vi.advanceTimersByTimeAsync(25000);

      // Connection should still exist (timer was cancelled)
      expect(pool.getConnection("pod-1")).toBeDefined();
      expect(pool.getConnection("pod-1")?.subscribers.size).toBe(1);
    });
  });

  describe("getStatus", () => {
    it("should return 'none' for unknown pod", () => {
      expect(pool.getStatus("unknown")).toBe("none");
    });
  });

  describe("isConnected", () => {
    it("should return false for unknown pod", () => {
      expect(pool.isConnected("unknown")).toBe(false);
    });
  });

  describe("isRunnerDisconnected", () => {
    it("should return false for unknown pod", () => {
      expect(pool.isRunnerDisconnected("unknown")).toBe(false);
    });
  });

  describe("disconnect", () => {
    it("should close connection and remove from pool", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await vi.runAllTimersAsync();

      pool.disconnect("pod-1");

      expect(pool.getStatus("pod-1")).toBe("none");
    });

    it("should be safe to call for non-existent pod", () => {
      expect(() => pool.disconnect("unknown")).not.toThrow();
    });
  });

  describe("disconnectAll", () => {
    it("should disconnect all connections", async () => {
      const onMessage = vi.fn();
      await pool.subscribe("pod-1", "sub-1", onMessage);
      await pool.subscribe("pod-2", "sub-2", onMessage);
      await vi.runAllTimersAsync();

      pool.disconnectAll();

      expect(pool.getStatus("pod-1")).toBe("none");
      expect(pool.getStatus("pod-2")).toBe("none");
    });
  });
});
