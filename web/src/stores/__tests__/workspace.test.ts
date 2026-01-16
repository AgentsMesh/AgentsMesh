import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { useWorkspaceStore, terminalPool } from "../workspace";

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = MockWebSocket.OPEN;
  binaryType = "arraybuffer";
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: unknown }) => void) | null = null;
  onerror: ((error: unknown) => void) | null = null;
  onclose: (() => void) | null = null;

  send = vi.fn();
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED;
  });
}

vi.stubGlobal("WebSocket", MockWebSocket);

describe("Workspace Store", () => {
  beforeEach(() => {
    localStorage.clear();
    // Reset store to initial state
    useWorkspaceStore.setState({
      panes: [],
      activePane: null,
      gridLayout: { type: "1x1", rows: 1, cols: 1 },
      mobileActiveIndex: 0,
      terminalFontSize: 14,
      _hasHydrated: false,
    });
  });

  describe("initial state", () => {
    it("should have default values", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      expect(result.current.panes).toEqual([]);
      expect(result.current.activePane).toBeNull();
      expect(result.current.gridLayout).toEqual({ type: "1x1", rows: 1, cols: 1 });
      expect(result.current.mobileActiveIndex).toBe(0);
      expect(result.current.terminalFontSize).toBe(14);
    });
  });

  describe("panes management", () => {
    it("should add a new pane", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let paneId: string;
      act(() => {
        paneId = result.current.addPane("pod-123");
      });

      expect(result.current.panes).toHaveLength(1);
      expect(result.current.panes[0].podKey).toBe("pod-123");
      expect(result.current.panes[0].isActive).toBe(true);
      expect(result.current.activePane).toBe(paneId!);
    });

    it("should use custom title when provided", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-123", "My Terminal");
      });

      expect(result.current.panes[0].title).toBe("My Terminal");
    });

    it("should generate default title from podKey", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-abc12345");
      });

      expect(result.current.panes[0].title).toBe("Pod pod-abc1");
    });

    it("should return existing pane id if pod already open", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let firstId: string;
      let secondId: string;
      act(() => {
        firstId = result.current.addPane("pod-123");
        secondId = result.current.addPane("pod-123");
      });

      expect(firstId!).toBe(secondId!);
      expect(result.current.panes).toHaveLength(1);
    });

    it("should remove a pane", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let paneId: string;
      act(() => {
        paneId = result.current.addPane("pod-123");
      });

      expect(result.current.panes).toHaveLength(1);

      act(() => {
        result.current.removePane(paneId!);
      });

      expect(result.current.panes).toHaveLength(0);
      expect(result.current.activePane).toBeNull();
    });

    it("should set next pane as active when active pane is removed", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let firstId: string;
      act(() => {
        firstId = result.current.addPane("pod-1");
        result.current.addPane("pod-2");
      });

      expect(result.current.panes).toHaveLength(2);
      expect(result.current.activePane).toBe(result.current.panes[1].id);

      act(() => {
        result.current.removePane(result.current.panes[1].id);
      });

      expect(result.current.panes).toHaveLength(1);
      expect(result.current.activePane).toBe(firstId!);
    });

    it("should clear all panes", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
        result.current.addPane("pod-3");
      });

      expect(result.current.panes).toHaveLength(3);

      act(() => {
        result.current.clearAllPanes();
      });

      expect(result.current.panes).toHaveLength(0);
      expect(result.current.activePane).toBeNull();
      expect(result.current.mobileActiveIndex).toBe(0);
    });
  });

  describe("active pane", () => {
    it("should set active pane", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let firstId: string;
      act(() => {
        firstId = result.current.addPane("pod-1");
        result.current.addPane("pod-2");
      });

      act(() => {
        result.current.setActivePane(firstId!);
      });

      expect(result.current.activePane).toBe(firstId!);
      expect(result.current.panes[0].isActive).toBe(true);
      expect(result.current.panes[1].isActive).toBe(false);
    });

    it("should set active pane to null", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.setActivePane(null);
      });

      expect(result.current.activePane).toBeNull();
    });
  });

  describe("grid layout", () => {
    it("should set grid layout", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.setGridLayout({ type: "2x2", rows: 2, cols: 2 });
      });

      expect(result.current.gridLayout).toEqual({ type: "2x2", rows: 2, cols: 2 });
    });

    it("should update pane position", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      let paneId: string;
      act(() => {
        paneId = result.current.addPane("pod-1");
      });

      act(() => {
        result.current.updatePanePosition(paneId!, { x: 1, y: 1, w: 2, h: 2 });
      });

      expect(result.current.panes[0].gridPosition).toEqual({ x: 1, y: 1, w: 2, h: 2 });
    });
  });

  describe("mobile state", () => {
    it("should set mobile active index", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
        result.current.addPane("pod-3");
      });

      act(() => {
        result.current.setMobileActiveIndex(1);
      });

      expect(result.current.mobileActiveIndex).toBe(1);
      expect(result.current.activePane).toBe(result.current.panes[1].id);
    });

    it("should not set invalid mobile active index", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-1");
        result.current.addPane("pod-2");
      });

      const initialIndex = result.current.mobileActiveIndex;

      act(() => {
        result.current.setMobileActiveIndex(99);
      });

      expect(result.current.mobileActiveIndex).toBe(initialIndex);
    });
  });

  describe("terminal settings", () => {
    it("should set terminal font size", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.setTerminalFontSize(16);
      });

      expect(result.current.terminalFontSize).toBe(16);
    });

    it("should clamp font size to minimum", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.setTerminalFontSize(5);
      });

      expect(result.current.terminalFontSize).toBe(10);
    });

    it("should clamp font size to maximum", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.setTerminalFontSize(50);
      });

      expect(result.current.terminalFontSize).toBe(24);
    });
  });

  describe("getPaneByPodKey", () => {
    it("should find pane by podKey", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.addPane("pod-123", "Test Terminal");
      });

      const pane = result.current.getPaneByPodKey("pod-123");
      expect(pane).toBeDefined();
      expect(pane?.podKey).toBe("pod-123");
      expect(pane?.title).toBe("Test Terminal");
    });

    it("should return undefined for non-existent podKey", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      const pane = result.current.getPaneByPodKey("non-existent");
      expect(pane).toBeUndefined();
    });
  });

  describe("hydration", () => {
    it("should set hydration state", () => {
      const { result } = renderHook(() => useWorkspaceStore());

      act(() => {
        result.current.setHasHydrated(true);
      });

      expect(result.current._hasHydrated).toBe(true);
    });
  });
});

// Track last created WebSocket instance
let lastMockWsInstance: MockWebSocket | null = null;

// Enhanced MockWebSocket that tracks instances
class TrackedMockWebSocket extends MockWebSocket {
  constructor() {
    super();
    // eslint-disable-next-line @typescript-eslint/no-this-alias
    lastMockWsInstance = this;
  }
}

describe("Terminal Connection Pool", () => {
  beforeEach(() => {
    // Clear all connections
    terminalPool.disconnectAll();
    localStorage.clear();
    lastMockWsInstance = null;

    // Set up auth data for WebSocket URL construction
    localStorage.setItem(
      "agentsmesh-auth",
      JSON.stringify({
        state: {
          token: "test-token",
          currentOrg: { slug: "test-org" },
        },
      })
    );

    // Use TrackedMockWebSocket
    vi.stubGlobal("WebSocket", TrackedMockWebSocket);
  });

  describe("getStatus", () => {
    it("should return 'none' for non-existent connection", () => {
      expect(terminalPool.getStatus("non-existent")).toBe("none");
    });
  });

  describe("isConnected", () => {
    it("should return false for non-existent connection", () => {
      expect(terminalPool.isConnected("non-existent")).toBe(false);
    });
  });

  describe("getConnection", () => {
    it("should return undefined for non-existent connection", () => {
      expect(terminalPool.getConnection("non-existent")).toBeUndefined();
    });
  });

  describe("connect", () => {
    it("should create a new WebSocket connection", () => {
      const onMessage = vi.fn();
      const { send, disconnect } = terminalPool.connect("pod-123", onMessage);

      expect(send).toBeDefined();
      expect(disconnect).toBeDefined();
      expect(terminalPool.getConnection("pod-123")).toBeDefined();
    });

    it("should reuse existing connection for same podKey", () => {
      const onMessage1 = vi.fn();
      const onMessage2 = vi.fn();

      terminalPool.connect("pod-123", onMessage1);
      terminalPool.connect("pod-123", onMessage2);

      // Both listeners should be added to the same connection
      const conn = terminalPool.getConnection("pod-123");
      expect(conn?.listeners.size).toBe(2);
    });

    it('should not replay buffer to new listener on existing connection (backend sends scrollback)', () => {
      const onMessage1 = vi.fn();
      terminalPool.connect("pod-123", onMessage1);

      // Simulate receiving a message
      const conn = terminalPool.getConnection("pod-123");
      if (conn) {
        const testData = new Uint8Array([1, 2, 3]);
        conn.buffer.push(testData);
      }

      // Add second listener - should NOT receive buffered message
      // (backend will send scrollback data on new WebSocket connection instead)
      const onMessage2 = vi.fn();
      terminalPool.connect("pod-123", onMessage2);

      expect(onMessage2).not.toHaveBeenCalled();
    });

    it("should handle WebSocket open event", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      // Simulate WebSocket open
      lastMockWsInstance!.onopen?.();

      const conn = terminalPool.getConnection("pod-123");
      expect(conn?.status).toBe("connected");
    });

    it("should handle WebSocket message event with ArrayBuffer", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      // Simulate WebSocket message with ArrayBuffer
      const testData = new ArrayBuffer(3);
      new Uint8Array(testData).set([1, 2, 3]);
      lastMockWsInstance!.onmessage?.({ data: testData });

      expect(onMessage).toHaveBeenCalled();
      const conn = terminalPool.getConnection("pod-123");
      expect(conn?.buffer.length).toBe(1);
    });

    it("should handle WebSocket message event with string", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      // Simulate WebSocket message with string
      lastMockWsInstance!.onmessage?.({ data: "test message" });

      expect(onMessage).toHaveBeenCalledWith("test message");
      // String messages should not be buffered
      const conn = terminalPool.getConnection("pod-123");
      expect(conn?.buffer.length).toBe(0);
    });

    it("should limit buffer size", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      // Add more than maxBufferSize messages
      for (let i = 0; i < 150; i++) {
        const testData = new ArrayBuffer(1);
        new Uint8Array(testData).set([i]);
        lastMockWsInstance!.onmessage?.({ data: testData });
      }

      const conn = terminalPool.getConnection("pod-123");
      expect(conn?.buffer.length).toBe(100); // maxBufferSize
    });

    it("should handle WebSocket error event", () => {
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      // Simulate WebSocket error
      lastMockWsInstance!.onerror?.(new Error("Connection failed"));

      const conn = terminalPool.getConnection("pod-123");
      expect(conn?.status).toBe("error");
      consoleSpy.mockRestore();
    });

    it("should handle WebSocket close event", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      // Simulate WebSocket close
      lastMockWsInstance!.onclose?.();

      const conn = terminalPool.getConnection("pod-123");
      expect(conn?.status).toBe("disconnected");
    });

    it("should handle missing auth data gracefully", () => {
      localStorage.clear();
      const onMessage = vi.fn();

      // Should not throw
      expect(() => terminalPool.connect("pod-123", onMessage)).not.toThrow();
    });

    it("should handle invalid auth data JSON", () => {
      localStorage.setItem("agentsmesh-auth", "invalid json");
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const onMessage = vi.fn();

      // Should not throw
      expect(() => terminalPool.connect("pod-123", onMessage)).not.toThrow();
      consoleSpy.mockRestore();
    });
  });

  describe("send", () => {
    it("should send data through WebSocket", () => {
      const onMessage = vi.fn();
      const { send } = terminalPool.connect("pod-123", onMessage);

      send("test input");

      expect(lastMockWsInstance!.send).toHaveBeenCalledWith(
        JSON.stringify({ type: "input", data: "test input" })
      );
    });

    it("should not send when WebSocket is not open", () => {
      const onMessage = vi.fn();
      const { send } = terminalPool.connect("pod-123", onMessage);

      lastMockWsInstance!.readyState = MockWebSocket.CLOSED;
      send("test input");

      expect(lastMockWsInstance!.send).not.toHaveBeenCalled();
    });
  });

  describe("sendResize", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("should send resize command through WebSocket after debounce", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      terminalPool.sendResize("pod-123", 24, 80);

      // Should not be called immediately due to debounce
      expect(lastMockWsInstance!.send).not.toHaveBeenCalled();

      // Advance timers past debounce period (150ms)
      vi.advanceTimersByTime(150);

      expect(lastMockWsInstance!.send).toHaveBeenCalledWith(
        JSON.stringify({ type: "resize", rows: 24, cols: 80 })
      );
    });

    it("should not send resize when WebSocket is not open", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      lastMockWsInstance!.readyState = MockWebSocket.CLOSED;
      terminalPool.sendResize("pod-123", 24, 80);

      // Advance timers past debounce period
      vi.advanceTimersByTime(150);

      expect(lastMockWsInstance!.send).not.toHaveBeenCalled();
    });

    it("should not send resize for non-existent connection", () => {
      terminalPool.sendResize("non-existent", 24, 80);
      // Advance timers past debounce period
      vi.advanceTimersByTime(150);
      // Should not throw, just silently fail
    });
  });

  describe("removeListener", () => {
    it("should remove listener from connection", () => {
      const onMessage = vi.fn();
      const { disconnect } = terminalPool.connect("pod-123", onMessage);

      disconnect();

      const conn = terminalPool.getConnection("pod-123");
      expect(conn?.listeners.has(onMessage)).toBe(false);
    });

    it("should do nothing for non-existent connection", () => {
      const onMessage = vi.fn();
      // Should not throw
      expect(() => {
        // Access private method through the returned disconnect function
        const { disconnect } = terminalPool.connect("pod-temp", onMessage);
        terminalPool.disconnectAll();
        disconnect(); // Try to disconnect after connection is gone
      }).not.toThrow();
    });
  });

  describe("disconnect", () => {
    it("should close WebSocket and remove connection", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      terminalPool.disconnect("pod-123");

      expect(lastMockWsInstance!.close).toHaveBeenCalled();
      expect(terminalPool.getConnection("pod-123")).toBeUndefined();
    });

    it("should not close WebSocket if already closed", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      lastMockWsInstance!.readyState = MockWebSocket.CLOSED;
      terminalPool.disconnect("pod-123");

      expect(lastMockWsInstance!.close).not.toHaveBeenCalled();
    });

    it("should do nothing for non-existent connection", () => {
      // Should not throw
      expect(() => terminalPool.disconnect("non-existent")).not.toThrow();
    });
  });

  describe("disconnectAll", () => {
    it("should disconnect all connections", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-1", onMessage);
      terminalPool.connect("pod-2", onMessage);

      terminalPool.disconnectAll();

      expect(terminalPool.getConnection("pod-1")).toBeUndefined();
      expect(terminalPool.getConnection("pod-2")).toBeUndefined();
    });
  });

  describe("isConnected", () => {
    it("should return true when connected and WebSocket is open", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);

      // Simulate open
      lastMockWsInstance!.onopen?.();

      expect(terminalPool.isConnected("pod-123")).toBe(true);
    });

    it("should return false when status is connected but WebSocket is not open", () => {
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);
      lastMockWsInstance!.onopen?.();

      lastMockWsInstance!.readyState = MockWebSocket.CLOSED;

      expect(terminalPool.isConnected("pod-123")).toBe(false);
    });

    it("should return false when status is error", () => {
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const onMessage = vi.fn();
      terminalPool.connect("pod-123", onMessage);
      lastMockWsInstance!.onerror?.(new Error("test"));

      expect(terminalPool.isConnected("pod-123")).toBe(false);
      consoleSpy.mockRestore();
    });
  });
});
