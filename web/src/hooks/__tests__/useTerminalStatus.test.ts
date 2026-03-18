import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

// Mock terminalPool
let statusCallback: ((info: { status: string; runnerDisconnected: boolean }) => void) | null = null;
const mockUnsubscribe = vi.fn();

vi.mock("@/stores/terminalConnection", () => ({
  terminalPool: {
    onStatusChange: vi.fn((podKey: string, listener: (info: { status: string; runnerDisconnected: boolean }) => void) => {
      statusCallback = listener;
      // Immediately call with initial status
      listener({ status: "none", runnerDisconnected: false });
      return mockUnsubscribe;
    }),
  },
}));

import { useTerminalStatus } from "../useTerminalStatus";

describe("useTerminalStatus", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    statusCallback = null;
  });

  it("returns initial status of none", () => {
    const { result } = renderHook(() => useTerminalStatus("pod-1"));
    expect(result.current.status).toBe("none");
    expect(result.current.runnerDisconnected).toBe(false);
  });

  it("subscribes to terminalPool on mount", async () => {
    const { terminalPool } = await import("@/stores/terminalConnection");
    renderHook(() => useTerminalStatus("pod-1"));
    expect(terminalPool.onStatusChange).toHaveBeenCalledWith("pod-1", expect.any(Function));
  });

  it("updates when status changes", () => {
    const { result } = renderHook(() => useTerminalStatus("pod-1"));

    act(() => {
      statusCallback?.({ status: "connected", runnerDisconnected: false });
    });

    expect(result.current.status).toBe("connected");
  });

  it("tracks runner disconnection", () => {
    const { result } = renderHook(() => useTerminalStatus("pod-1"));

    act(() => {
      statusCallback?.({ status: "connected", runnerDisconnected: true });
    });

    expect(result.current.runnerDisconnected).toBe(true);
  });

  it("unsubscribes on unmount", () => {
    const { unmount } = renderHook(() => useTerminalStatus("pod-1"));
    unmount();
    expect(mockUnsubscribe).toHaveBeenCalled();
  });

  it("resubscribes when podKey changes", async () => {
    const { terminalPool } = await import("@/stores/terminalConnection");
    const { rerender } = renderHook(
      ({ podKey }) => useTerminalStatus(podKey),
      { initialProps: { podKey: "pod-1" } }
    );

    rerender({ podKey: "pod-2" });

    expect(terminalPool.onStatusChange).toHaveBeenCalledWith("pod-2", expect.any(Function));
    expect(mockUnsubscribe).toHaveBeenCalled();
  });
});
