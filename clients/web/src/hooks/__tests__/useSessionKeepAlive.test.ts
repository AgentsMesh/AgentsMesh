import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook } from "@testing-library/react";

const { mockRefreshSession } = vi.hoisted(() => ({ mockRefreshSession: vi.fn() }));

vi.mock("@/stores/auth", () => ({
  useAuthStore: (selector: (s: { refreshSession: () => Promise<void> }) => unknown) =>
    selector({ refreshSession: mockRefreshSession }),
}));

import { useSessionKeepAlive } from "../useSessionKeepAlive";

const INTERVAL_MS = 30 * 60 * 1000;

function setVisibility(state: DocumentVisibilityState) {
  Object.defineProperty(document, "visibilityState", { value: state, configurable: true });
}

describe("useSessionKeepAlive", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockRefreshSession.mockReset().mockResolvedValue(undefined);
    setVisibility("visible");
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("refreshes when the window regains focus", () => {
    renderHook(() => useSessionKeepAlive());
    window.dispatchEvent(new Event("focus"));
    expect(mockRefreshSession).toHaveBeenCalledTimes(1);
  });

  it("refreshes when the document becomes visible", () => {
    renderHook(() => useSessionKeepAlive());
    setVisibility("visible");
    document.dispatchEvent(new Event("visibilitychange"));
    expect(mockRefreshSession).toHaveBeenCalledTimes(1);
  });

  it("does not refresh while the document is hidden", () => {
    renderHook(() => useSessionKeepAlive());
    setVisibility("hidden");
    document.dispatchEvent(new Event("visibilitychange"));
    expect(mockRefreshSession).not.toHaveBeenCalled();
  });

  it("refreshes on the interval while foregrounded", () => {
    renderHook(() => useSessionKeepAlive());
    vi.advanceTimersByTime(INTERVAL_MS);
    expect(mockRefreshSession).toHaveBeenCalledTimes(1);
  });

  it("stops refreshing after unmount", () => {
    const { unmount } = renderHook(() => useSessionKeepAlive());
    unmount();
    window.dispatchEvent(new Event("focus"));
    vi.advanceTimersByTime(INTERVAL_MS);
    expect(mockRefreshSession).not.toHaveBeenCalled();
  });

  it("swallows a rejected refresh so it never throws", () => {
    mockRefreshSession.mockRejectedValue(new Error("network down"));
    renderHook(() => useSessionKeepAlive());
    expect(() => window.dispatchEvent(new Event("focus"))).not.toThrow();
  });
});
