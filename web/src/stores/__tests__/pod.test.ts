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

describe("Pod Store — basic reads", () => {
  beforeEach(resetPodStore);

  describe("initial state", () => {
    it("should have default values", () => {
      const state = usePodStore.getState();
      expect(state.pods).toEqual([]);
      expect(state.currentPod).toBeNull();
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
    });
  });

  describe("fetchPods", () => {
    it("should fetch pods successfully", async () => {
      vi.mocked(podApi.list).mockResolvedValue({
        pods: [mockPod, mockPod2], total: 2, limit: 20, offset: 0,
      });

      await act(async () => { await usePodStore.getState().fetchPods(); });

      const state = usePodStore.getState();
      expect(state.pods).toHaveLength(2);
      expect(state.pods[0].pod_key).toBe("pod-abc-123");
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
    });

    it("should pass filters to API", async () => {
      vi.mocked(podApi.list).mockResolvedValue({ pods: [], total: 0, limit: 20, offset: 0 });

      await act(async () => {
        await usePodStore.getState().fetchPods({ status: "running", runnerId: 1 });
      });

      expect(podApi.list).toHaveBeenCalledWith({ status: "running", runnerId: 1 });
    });

    it("should handle empty response", async () => {
      vi.mocked(podApi.list).mockResolvedValue({
        pods: undefined as unknown as typeof mockPod[], total: 0, limit: 20, offset: 0,
      });

      await act(async () => { await usePodStore.getState().fetchPods(); });
      expect(usePodStore.getState().pods).toEqual([]);
    });

    it("should handle fetch error", async () => {
      vi.mocked(podApi.list).mockRejectedValue({ message: "Network error" });

      await act(async () => { await usePodStore.getState().fetchPods(); });

      const state = usePodStore.getState();
      expect(state.error).toBe("Network error");
      expect(state.loading).toBe(false);
    });

    it("should use default error message when no message provided", async () => {
      vi.mocked(podApi.list).mockRejectedValue({});

      await act(async () => { await usePodStore.getState().fetchPods(); });
      expect(usePodStore.getState().error).toBe("Failed to fetch pods");
    });
  });

  describe("fetchPod", () => {
    it("should fetch single pod successfully", async () => {
      vi.mocked(podApi.get).mockResolvedValue({ pod: mockPod });

      await act(async () => { await usePodStore.getState().fetchPod("pod-abc-123"); });

      const state = usePodStore.getState();
      expect(state.currentPod).toBeNull();
      expect(state.loading).toBe(false);
    });

    it("should add fetched pod to pods array when not present", async () => {
      vi.mocked(podApi.get).mockResolvedValue({ pod: mockPod });
      expect(usePodStore.getState().pods).toEqual([]);

      await act(async () => { await usePodStore.getState().fetchPod("pod-abc-123"); });

      const state = usePodStore.getState();
      expect(state.pods).toHaveLength(1);
      expect(state.pods[0]).toEqual(mockPod);
    });

    it("should update existing pod in pods array when present", async () => {
      const updatedPod = { ...mockPod, status: "terminated" as const };
      vi.mocked(podApi.get).mockResolvedValue({ pod: updatedPod });
      usePodStore.setState({ pods: [mockPod, mockPod2] });

      await act(async () => { await usePodStore.getState().fetchPod("pod-abc-123"); });

      const state = usePodStore.getState();
      expect(state.pods).toHaveLength(2);
      expect(state.pods.find(p => p.pod_key === "pod-abc-123")?.status).toBe("terminated");
      expect(state.pods.find(p => p.pod_key === "pod-def-456")).toEqual(mockPod2);
    });

    it("should handle fetch error", async () => {
      vi.mocked(podApi.get).mockRejectedValue({ message: "Pod not found" });

      await act(async () => {
        await usePodStore.getState().fetchPod("non-existent").catch(() => {});
      });

      const state = usePodStore.getState();
      expect(state.error).toBeNull();
      expect(state.loading).toBe(false);
    });
  });

  describe("setCurrentPod", () => {
    it("should set current pod", () => {
      act(() => { usePodStore.getState().setCurrentPod(mockPod); });
      expect(usePodStore.getState().currentPod).toEqual(mockPod);
    });

    it("should set to null", () => {
      usePodStore.setState({ currentPod: mockPod });
      act(() => { usePodStore.getState().setCurrentPod(null); });
      expect(usePodStore.getState().currentPod).toBeNull();
    });
  });

  describe("clearError", () => {
    it("should clear error", () => {
      usePodStore.setState({ error: "Some error" });
      act(() => { usePodStore.getState().clearError(); });
      expect(usePodStore.getState().error).toBeNull();
    });
  });
});
