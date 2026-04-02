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

import { podApi, ApiError } from "@/lib/api";

describe("Pod Store — terminatePod", () => {
  beforeEach(resetPodStore);

  it("should terminate pod and update local state", async () => {
    usePodStore.setState({ pods: [mockPod, mockPod2] });
    vi.mocked(podApi.terminate).mockResolvedValue({ message: "ok" });

    await act(async () => {
      await usePodStore.getState().terminatePod("pod-abc-123");
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.status).toBe("terminated");
    // Other pods unaffected
    expect(usePodStore.getState().pods.find(p => p.pod_key === "pod-def-456")?.status).toBe("running");
  });

  it("should treat 404 as already terminated", async () => {
    usePodStore.setState({ pods: [mockPod] });
    vi.mocked(podApi.terminate).mockRejectedValue(
      new ApiError(404, "Not Found")
    );

    // Should NOT throw
    await act(async () => {
      await usePodStore.getState().terminatePod("pod-abc-123");
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.status).toBe("terminated");
    expect(usePodStore.getState().error).toBeNull();
  });

  it("should rethrow non-404 errors", async () => {
    usePodStore.setState({ pods: [mockPod] });
    vi.mocked(podApi.terminate).mockRejectedValue(new Error("Server error"));

    await act(async () => {
      await expect(
        usePodStore.getState().terminatePod("pod-abc-123")
      ).rejects.toThrow("Server error");
    });

    expect(usePodStore.getState().error).toBe("Server error");
  });

  it("should update currentPod when terminating active pod", async () => {
    usePodStore.setState({ pods: [mockPod], currentPod: mockPod });
    vi.mocked(podApi.terminate).mockResolvedValue({ message: "ok" });

    await act(async () => {
      await usePodStore.getState().terminatePod("pod-abc-123");
    });

    expect(usePodStore.getState().currentPod?.status).toBe("terminated");
  });

  it("should be no-op for non-existent pod in local state", async () => {
    usePodStore.setState({ pods: [mockPod] });
    vi.mocked(podApi.terminate).mockResolvedValue({ message: "ok" });

    await act(async () => {
      await usePodStore.getState().terminatePod("non-existent");
    });

    // Pods unchanged
    expect(usePodStore.getState().pods).toHaveLength(1);
    expect(usePodStore.getState().pods[0].status).toBe("running");
  });
});
