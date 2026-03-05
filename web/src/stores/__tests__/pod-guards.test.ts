import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { usePodStore } from "../pod";
import { mockPod, mockPod2, resetPodStore } from "./pod-test-utils";

vi.mock("@/lib/api", () => ({
  podApi: { list: vi.fn(), get: vi.fn(), create: vi.fn(), terminate: vi.fn() },
  ApiError: class extends Error {
    status: number;
    statusText: string;
    constructor(s: number, t: string) { super(`API Error: ${s} ${t}`); this.name = "ApiError"; this.status = s; this.statusText = t; }
  },
}));

import { podApi } from "@/lib/api";

describe("Pod Store — fetchSidebarPods", () => {
  beforeEach(resetPodStore);

  it("should fetch with correct status mapping for running filter", async () => {
    vi.mocked(podApi.list).mockResolvedValue({ pods: [mockPod], total: 1, limit: 20, offset: 0 });

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("running");
    });

    expect(podApi.list).toHaveBeenCalledWith({
      status: "running,initializing",
      limit: 20,
      offset: 0,
    });
    expect(usePodStore.getState().currentSidebarFilter).toBe("running");
  });

  it("should fetch with completed status mapping", async () => {
    vi.mocked(podApi.list).mockResolvedValue({ pods: [], total: 0, limit: 20, offset: 0 });

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("completed");
    });

    expect(podApi.list).toHaveBeenCalledWith({
      status: "terminated,failed,paused,completed,error,orphaned",
      limit: 20,
      offset: 0,
    });
  });

  it("should set loading during fetch and clear after", async () => {
    let loadingDuringFetch = false;
    vi.mocked(podApi.list).mockImplementation(async () => {
      loadingDuringFetch = usePodStore.getState().loading;
      return { pods: [], total: 0, limit: 20, offset: 0 };
    });

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("running");
    });

    expect(loadingDuringFetch).toBe(true);
    expect(usePodStore.getState().loading).toBe(false);
  });

  it("should compute podHasMore correctly", async () => {
    vi.mocked(podApi.list).mockResolvedValue({ pods: [mockPod], total: 5, limit: 20, offset: 0 });

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("running");
    });

    expect(usePodStore.getState().podHasMore).toBe(true);
    expect(usePodStore.getState().podTotal).toBe(5);
  });

  it("should handle error and clear loading", async () => {
    vi.mocked(podApi.list).mockRejectedValue(new Error("Network error"));

    await act(async () => {
      await usePodStore.getState().fetchSidebarPods("running");
    });

    expect(usePodStore.getState().error).toBe("Network error");
    expect(usePodStore.getState().loading).toBe(false);
  });
});

describe("Pod Store — loadMorePods", () => {
  beforeEach(resetPodStore);

  it("should load more pods with correct offset", async () => {
    usePodStore.setState({
      pods: [mockPod],
      podHasMore: true,
      currentSidebarFilter: "running",
    });
    vi.mocked(podApi.list).mockResolvedValue({ pods: [mockPod2], total: 2, limit: 20, offset: 1 });

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    expect(podApi.list).toHaveBeenCalledWith({
      status: "running,initializing",
      limit: 20,
      offset: 1,
    });
    expect(usePodStore.getState().pods).toHaveLength(2);
  });

  it("should skip when no more pods", async () => {
    usePodStore.setState({ pods: [mockPod], podHasMore: false });

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    expect(podApi.list).not.toHaveBeenCalled();
  });

  it("should skip when already loading more", async () => {
    usePodStore.setState({ pods: [mockPod], podHasMore: true, loadingMore: true });

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    expect(podApi.list).not.toHaveBeenCalled();
  });

  it("should deduplicate pods already in list", async () => {
    usePodStore.setState({
      pods: [mockPod, mockPod2],
      podHasMore: true,
      currentSidebarFilter: "running",
    });
    // API returns mockPod2 again (realtime event already added it)
    vi.mocked(podApi.list).mockResolvedValue({ pods: [mockPod2], total: 3, limit: 20, offset: 2 });

    await act(async () => {
      await usePodStore.getState().loadMorePods();
    });

    // Should not duplicate mockPod2
    expect(usePodStore.getState().pods).toHaveLength(2);
  });
});

describe("Pod Store — timestamp guards", () => {
  beforeEach(resetPodStore);

  it("should reject stale updatePodStatus with older timestamp", () => {
    usePodStore.setState({
      pods: [mockPod],
      podTimestamps: { "pod-abc-123": 2000 },
    });

    act(() => {
      usePodStore.getState().updatePodStatus("pod-abc-123", "terminated", undefined, undefined, undefined, 1000);
    });

    // Stale update rejected — status unchanged
    expect(usePodStore.getState().pods[0].status).toBe("running");
  });

  it("should accept updatePodStatus with newer timestamp", () => {
    usePodStore.setState({
      pods: [mockPod],
      podTimestamps: { "pod-abc-123": 1000 },
    });

    act(() => {
      usePodStore.getState().updatePodStatus("pod-abc-123", "terminated", undefined, undefined, undefined, 2000);
    });

    expect(usePodStore.getState().pods[0].status).toBe("terminated");
    expect(usePodStore.getState().podTimestamps["pod-abc-123"]).toBe(2000);
  });

  it("should accept update without timestamp (backwards compat)", () => {
    usePodStore.setState({
      pods: [mockPod],
      podTimestamps: { "pod-abc-123": 5000 },
    });

    act(() => {
      usePodStore.getState().updatePodStatus("pod-abc-123", "paused");
    });

    // No timestamp = always accepted
    expect(usePodStore.getState().pods[0].status).toBe("paused");
  });

  it("should reject stale updateAgentStatus", () => {
    usePodStore.setState({
      pods: [mockPod],
      podTimestamps: { "pod-abc-123": 3000 },
    });

    act(() => {
      usePodStore.getState().updateAgentStatus("pod-abc-123", "idle", 1000);
    });

    expect(usePodStore.getState().pods[0].agent_status).toBe("executing");
  });

  it("should reject stale updatePodTitle", () => {
    const titledPod = { ...mockPod, title: "Original" };
    usePodStore.setState({
      pods: [titledPod],
      podTimestamps: { "pod-abc-123": 3000 },
    });

    act(() => {
      usePodStore.getState().updatePodTitle("pod-abc-123", "Stale Title", 1000);
    });

    expect(usePodStore.getState().pods[0].title).toBe("Original");
  });
});
