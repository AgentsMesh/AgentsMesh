import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { usePodStore } from "../pod";
import { mockPod, mockPod2, resetPodStore } from "./pod-test-utils";

vi.mock("@/lib/api", () => ({
  podApi: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    terminate: vi.fn(),
    updateAlias: vi.fn(),
    updatePerpetual: vi.fn(),
  },
  ApiError: class extends Error {
    status: number;
    statusText: string;
    constructor(s: number, t: string) { super(`API Error: ${s} ${t}`); this.name = "ApiError"; this.status = s; this.statusText = t; }
  },
}));

import { podApi } from "@/lib/api";

describe("Pod Store — updatePodPerpetual", () => {
  beforeEach(resetPodStore);

  it("should optimistically set perpetual=true", async () => {
    usePodStore.setState({ pods: [{ ...mockPod, perpetual: false }] });
    vi.mocked(podApi.updatePerpetual).mockResolvedValue({ message: "ok" });

    await act(async () => {
      await usePodStore.getState().updatePodPerpetual("pod-abc-123", true);
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.perpetual).toBe(true);
  });

  it("should optimistically set perpetual=false", async () => {
    usePodStore.setState({ pods: [{ ...mockPod, perpetual: true }] });
    vi.mocked(podApi.updatePerpetual).mockResolvedValue({ message: "ok" });

    await act(async () => {
      await usePodStore.getState().updatePodPerpetual("pod-abc-123", false);
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.perpetual).toBe(false);
  });

  it("should revert on API failure", async () => {
    usePodStore.setState({ pods: [{ ...mockPod, perpetual: false }] });
    vi.mocked(podApi.updatePerpetual).mockRejectedValue(new Error("Server error"));
    vi.mocked(podApi.get).mockResolvedValue({ pod: { ...mockPod, perpetual: false } });

    await act(async () => {
      await expect(
        usePodStore.getState().updatePodPerpetual("pod-abc-123", true)
      ).rejects.toThrow("Server error");
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.perpetual).toBe(false);
  });

  it("should update currentPod when it matches", async () => {
    const pod = { ...mockPod, perpetual: false };
    usePodStore.setState({ pods: [pod], currentPod: pod });
    vi.mocked(podApi.updatePerpetual).mockResolvedValue({ message: "ok" });

    await act(async () => {
      await usePodStore.getState().updatePodPerpetual("pod-abc-123", true);
    });

    expect(usePodStore.getState().currentPod?.perpetual).toBe(true);
  });

  it("should not affect other pods", async () => {
    usePodStore.setState({ pods: [{ ...mockPod, perpetual: false }, mockPod2] });
    vi.mocked(podApi.updatePerpetual).mockResolvedValue({ message: "ok" });

    await act(async () => {
      await usePodStore.getState().updatePodPerpetual("pod-abc-123", true);
    });

    expect(usePodStore.getState().pods.find(p => p.pod_key === "pod-def-456")?.perpetual).toBeUndefined();
  });
});

describe("Pod Store — updatePodPerpetualFromEvent", () => {
  beforeEach(resetPodStore);

  it("should update perpetual field on existing pod", () => {
    usePodStore.setState({ pods: [{ ...mockPod, perpetual: false }] });

    act(() => {
      usePodStore.getState().updatePodPerpetualFromEvent("pod-abc-123", true);
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.perpetual).toBe(true);
  });

  it("should sync currentPod when matching", () => {
    const pod = { ...mockPod, perpetual: false };
    usePodStore.setState({ pods: [pod], currentPod: pod });

    act(() => {
      usePodStore.getState().updatePodPerpetualFromEvent("pod-abc-123", true);
    });

    expect(usePodStore.getState().currentPod?.perpetual).toBe(true);
  });

  it("should skip non-existent pod", () => {
    usePodStore.setState({ pods: [mockPod] });

    act(() => {
      usePodStore.getState().updatePodPerpetualFromEvent("non-existent", true);
    });

    expect(usePodStore.getState().pods).toHaveLength(1);
    expect(usePodStore.getState().pods[0].perpetual).toBeUndefined();
  });
});
