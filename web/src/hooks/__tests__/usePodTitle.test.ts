import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";

// Mock pod store
let mockPods: Array<{ pod_key: string; name?: string }> = [];

vi.mock("@/stores/pod", () => ({
  usePodStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state = { pods: mockPods };
    return selector ? selector(state) : state;
  },
}));

vi.mock("@/lib/pod-utils", () => ({
  getPodDisplayName: (pod: { name?: string; pod_key: string }) => pod.name || pod.pod_key,
}));

// Import after mocks
import { usePodTitle } from "../usePodTitle";

describe("usePodTitle", () => {
  beforeEach(() => {
    mockPods = [
      { pod_key: "pod-abc123", name: "My Pod" },
      { pod_key: "pod-def456" },
    ];
  });

  it("returns pod display name when pod exists with name", () => {
    const { result } = renderHook(() => usePodTitle("pod-abc123"));
    expect(result.current).toBe("My Pod");
  });

  it("returns pod_key as display name when pod has no name", () => {
    const { result } = renderHook(() => usePodTitle("pod-def456"));
    expect(result.current).toBe("pod-def456");
  });

  it("falls back to truncated podKey when pod not found", () => {
    const { result } = renderHook(() => usePodTitle("unknown-pod-key-12345"));
    expect(result.current).toBe("unknown-");
  });

  it("uses custom fallback when pod not found", () => {
    const { result } = renderHook(() => usePodTitle("unknown-key", "Terminal"));
    expect(result.current).toBe("Terminal");
  });
});
