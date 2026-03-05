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

describe("Pod Store — updatePodStatus", () => {
  beforeEach(resetPodStore);

  it("should update status of existing pod", () => {
    usePodStore.setState({ pods: [mockPod, mockPod2] });

    act(() => {
      usePodStore.getState().updatePodStatus("pod-abc-123", "terminated");
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.status).toBe("terminated");
    // Other pod untouched
    expect(usePodStore.getState().pods.find(p => p.pod_key === "pod-def-456")?.status).toBe("running");
  });

  it("should update agent_status alongside status", () => {
    usePodStore.setState({ pods: [mockPod] });

    act(() => {
      usePodStore.getState().updatePodStatus("pod-abc-123", "running", "waiting");
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.agent_status).toBe("waiting");
  });

  it("should set error fields when status is error", () => {
    usePodStore.setState({ pods: [mockPod] });

    act(() => {
      usePodStore.getState().updatePodStatus("pod-abc-123", "error", undefined, "E001", "Timeout");
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.error_code).toBe("E001");
    expect(pod?.error_message).toBe("Timeout");
  });

  it("should clear error fields when transitioning away from error", () => {
    const errorPod = { ...mockPod, status: "error" as const, error_code: "E001", error_message: "Timeout" };
    usePodStore.setState({ pods: [errorPod] });

    act(() => {
      usePodStore.getState().updatePodStatus("pod-abc-123", "running");
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.error_code).toBeUndefined();
    expect(pod?.error_message).toBeUndefined();
  });

  it("should skip update for non-existent pod", () => {
    usePodStore.setState({ pods: [mockPod] });

    act(() => {
      usePodStore.getState().updatePodStatus("non-existent", "terminated");
    });

    // No change
    expect(usePodStore.getState().pods).toHaveLength(1);
    expect(usePodStore.getState().pods[0].status).toBe("running");
  });

  it("should update currentPod when it matches", () => {
    usePodStore.setState({ pods: [mockPod], currentPod: mockPod });

    act(() => {
      usePodStore.getState().updatePodStatus("pod-abc-123", "paused");
    });

    expect(usePodStore.getState().currentPod?.status).toBe("paused");
  });

  it("should not update currentPod when it differs", () => {
    usePodStore.setState({ pods: [mockPod, mockPod2], currentPod: mockPod2 });

    act(() => {
      usePodStore.getState().updatePodStatus("pod-abc-123", "paused");
    });

    expect(usePodStore.getState().currentPod?.pod_key).toBe("pod-def-456");
    expect(usePodStore.getState().currentPod?.status).toBe("running");
  });
});

describe("Pod Store — updateAgentStatus", () => {
  beforeEach(resetPodStore);

  it("should update agent_status only", () => {
    usePodStore.setState({ pods: [mockPod] });

    act(() => {
      usePodStore.getState().updateAgentStatus("pod-abc-123", "idle");
    });

    const pod = usePodStore.getState().pods.find(p => p.pod_key === "pod-abc-123");
    expect(pod?.agent_status).toBe("idle");
    expect(pod?.status).toBe("running"); // unchanged
  });

  it("should skip for non-existent pod", () => {
    usePodStore.setState({ pods: [mockPod] });

    act(() => {
      usePodStore.getState().updateAgentStatus("non-existent", "idle");
    });

    expect(usePodStore.getState().pods[0].agent_status).toBe("executing");
  });

  it("should sync currentPod when matching", () => {
    usePodStore.setState({ pods: [mockPod], currentPod: mockPod });

    act(() => {
      usePodStore.getState().updateAgentStatus("pod-abc-123", "waiting");
    });

    expect(usePodStore.getState().currentPod?.agent_status).toBe("waiting");
  });
});

describe("Pod Store — updatePodTitle", () => {
  beforeEach(resetPodStore);

  it("should update title", () => {
    usePodStore.setState({ pods: [mockPod] });

    act(() => {
      usePodStore.getState().updatePodTitle("pod-abc-123", "New Title");
    });

    expect(usePodStore.getState().pods[0].title).toBe("New Title");
  });

  it("should skip for non-existent pod", () => {
    usePodStore.setState({ pods: [mockPod] });

    act(() => {
      usePodStore.getState().updatePodTitle("non-existent", "Title");
    });

    expect(usePodStore.getState().pods[0].title).toBeUndefined();
  });
});

describe("Pod Store — initProgress", () => {
  beforeEach(resetPodStore);

  it("should set init progress", () => {
    act(() => {
      usePodStore.getState().updatePodInitProgress("pod-abc-123", "cloning", 50, "Cloning repo...");
    });

    const progress = usePodStore.getState().initProgress["pod-abc-123"];
    expect(progress).toEqual({ phase: "cloning", progress: 50, message: "Cloning repo..." });
  });

  it("should update existing progress", () => {
    usePodStore.setState({
      initProgress: { "pod-abc-123": { phase: "cloning", progress: 50, message: "..." } },
    });

    act(() => {
      usePodStore.getState().updatePodInitProgress("pod-abc-123", "installing", 80, "Installing deps...");
    });

    expect(usePodStore.getState().initProgress["pod-abc-123"]).toEqual({
      phase: "installing", progress: 80, message: "Installing deps...",
    });
  });

  it("should clear init progress for specific pod", () => {
    usePodStore.setState({
      initProgress: {
        "pod-abc-123": { phase: "done", progress: 100, message: "Done" },
        "pod-def-456": { phase: "cloning", progress: 20, message: "..." },
      },
    });

    act(() => {
      usePodStore.getState().clearInitProgress("pod-abc-123");
    });

    const progress = usePodStore.getState().initProgress;
    expect(progress["pod-abc-123"]).toBeUndefined();
    expect(progress["pod-def-456"]).toBeDefined();
  });
});
