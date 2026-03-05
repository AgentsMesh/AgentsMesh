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

describe("Pod Store — createPod", () => {
  beforeEach(resetPodStore);

  it("should create pod and prepend to list", async () => {
    usePodStore.setState({ pods: [mockPod2] });
    const newPod = { ...mockPod, id: 3, pod_key: "pod-new-789" };
    vi.mocked(podApi.create).mockResolvedValue({ message: "ok", pod: newPod });

    let result: typeof mockPod;
    await act(async () => {
      result = await usePodStore.getState().createPod({ runnerId: 1 });
    });

    expect(result!.pod_key).toBe("pod-new-789");
    const state = usePodStore.getState();
    // New pod should be prepended
    expect(state.pods[0].pod_key).toBe("pod-new-789");
    expect(state.pods[1].pod_key).toBe("pod-def-456");
    expect(state.currentPod).toEqual(newPod);
  });

  it("should convert camelCase params to snake_case", async () => {
    vi.mocked(podApi.create).mockResolvedValue({ message: "ok", pod: mockPod });

    await act(async () => {
      await usePodStore.getState().createPod({
        runnerId: 1,
        agentTypeId: 2,
        repositoryId: 3,
        ticketSlug: "TICK-1",
        initialPrompt: "hello",
        branchName: "feat/x",
      });
    });

    expect(podApi.create).toHaveBeenCalledWith({
      runner_id: 1,
      agent_type_id: 2,
      repository_id: 3,
      ticket_slug: "TICK-1",
      initial_prompt: "hello",
      branch_name: "feat/x",
    });
  });

  it("should not set loading flag", async () => {
    let loadingDuringCreate = false;
    vi.mocked(podApi.create).mockImplementation(async () => {
      loadingDuringCreate = usePodStore.getState().loading;
      return { message: "ok", pod: mockPod };
    });

    await act(async () => {
      await usePodStore.getState().createPod({ runnerId: 1 });
    });

    expect(loadingDuringCreate).toBe(false);
  });

  it("should set error and rethrow on failure", async () => {
    vi.mocked(podApi.create).mockRejectedValue(new Error("Create failed"));

    await act(async () => {
      await expect(
        usePodStore.getState().createPod({ runnerId: 1 })
      ).rejects.toThrow("Create failed");
    });

    expect(usePodStore.getState().error).toBe("Create failed");
  });

  it("should not duplicate pod when already in list", async () => {
    usePodStore.setState({ pods: [mockPod] });
    const updated = { ...mockPod, status: "running" as const };
    vi.mocked(podApi.create).mockResolvedValue({ message: "ok", pod: updated });

    await act(async () => {
      await usePodStore.getState().createPod({ runnerId: 1 });
    });

    expect(usePodStore.getState().pods).toHaveLength(1);
  });
});

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
