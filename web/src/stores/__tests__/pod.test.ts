import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { usePodStore, Pod } from "../pod";

// Mock the pod API with inline class definition
vi.mock("@/lib/api", () => {
  // Define MockApiError class inline to avoid hoisting issues
  class MockApiError extends Error {
    status: number;
    constructor(message: string, status: number) {
      super(message);
      this.name = "ApiError";
      this.status = status;
    }
  }

  return {
    podApi: {
      list: vi.fn(),
      get: vi.fn(),
      create: vi.fn(),
      terminate: vi.fn(),
    },
    ApiError: MockApiError,
  };
});

import { podApi, ApiError } from "@/lib/api";
const MockApiError = ApiError as unknown as new (message: string, status: number) => Error & { status: number };

const mockPod: Pod = {
  id: 1,
  pod_key: "pod-abc-123",
  status: "running",
  agent_status: "executing",
  created_at: "2024-01-01T00:00:00Z",
  runner: {
    id: 1,
    node_id: "runner-1",
    status: "online",
  },
};

const mockPod2: Pod = {
  id: 2,
  pod_key: "pod-def-456",
  status: "running",
  agent_status: "waiting",
  created_at: "2024-01-02T00:00:00Z",
  runner: {
    id: 1,
    node_id: "runner-1",
    status: "online",
  },
};

describe("Pod Store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset store to initial state
    usePodStore.setState({
      pods: [],
      currentPod: null,
      loading: false,
      error: null,
    });
  });

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
        pods: [mockPod, mockPod2],
        total: 2,
      });

      await act(async () => {
        await usePodStore.getState().fetchPods();
      });

      const state = usePodStore.getState();
      expect(state.pods).toHaveLength(2);
      expect(state.pods[0].pod_key).toBe("pod-abc-123");
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
    });

    it("should pass filters to API", async () => {
      vi.mocked(podApi.list).mockResolvedValue({ pods: [], total: 0 });

      await act(async () => {
        await usePodStore.getState().fetchPods({ status: "running", runnerId: 1 });
      });

      expect(podApi.list).toHaveBeenCalledWith({
        status: "running",
        runnerId: 1,
      });
    });

    it("should handle empty response", async () => {
      vi.mocked(podApi.list).mockResolvedValue({ pods: undefined as unknown as typeof mockPod[], total: 0 });

      await act(async () => {
        await usePodStore.getState().fetchPods();
      });

      const state = usePodStore.getState();
      expect(state.pods).toEqual([]);
    });

    it("should handle fetch error", async () => {
      vi.mocked(podApi.list).mockRejectedValue({ message: "Network error" });

      await act(async () => {
        await usePodStore.getState().fetchPods();
      });

      const state = usePodStore.getState();
      expect(state.error).toBe("Network error");
      expect(state.loading).toBe(false);
    });

    it("should use default error message when no message provided", async () => {
      vi.mocked(podApi.list).mockRejectedValue({});

      await act(async () => {
        await usePodStore.getState().fetchPods();
      });

      const state = usePodStore.getState();
      expect(state.error).toBe("Failed to fetch pods");
    });
  });

  describe("fetchPod", () => {
    it("should fetch single pod successfully", async () => {
      vi.mocked(podApi.get).mockResolvedValue({ pod: mockPod });

      await act(async () => {
        await usePodStore.getState().fetchPod("pod-abc-123");
      });

      const state = usePodStore.getState();
      expect(state.currentPod).toEqual(mockPod);
      expect(state.loading).toBe(false);
    });

    it("should add fetched pod to pods array when not present", async () => {
      vi.mocked(podApi.get).mockResolvedValue({ pod: mockPod });

      // Start with empty pods array
      expect(usePodStore.getState().pods).toEqual([]);

      await act(async () => {
        await usePodStore.getState().fetchPod("pod-abc-123");
      });

      const state = usePodStore.getState();
      expect(state.pods).toHaveLength(1);
      expect(state.pods[0]).toEqual(mockPod);
    });

    it("should update existing pod in pods array when present", async () => {
      const updatedPod = { ...mockPod, status: "terminated" as const };
      vi.mocked(podApi.get).mockResolvedValue({ pod: updatedPod });

      // Start with existing pod
      usePodStore.setState({ pods: [mockPod, mockPod2] });

      await act(async () => {
        await usePodStore.getState().fetchPod("pod-abc-123");
      });

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
      expect(state.error).toBe("Pod not found");
      expect(state.loading).toBe(false);
    });
  });

  describe("createPod", () => {
    it("should create pod successfully", async () => {
      vi.mocked(podApi.create).mockResolvedValue({ pod: mockPod, message: "success" });

      let result: Pod;
      await act(async () => {
        result = await usePodStore.getState().createPod({
          runnerId: 1,
          agentTypeId: 1,
        });
      });

      const state = usePodStore.getState();
      expect(result!).toEqual(mockPod);
      expect(state.pods).toContainEqual(mockPod);
      expect(state.currentPod).toEqual(mockPod);
      expect(state.loading).toBe(false);
    });

    it("should convert camelCase to snake_case for API", async () => {
      vi.mocked(podApi.create).mockResolvedValue({ pod: mockPod, message: "success" });

      await act(async () => {
        await usePodStore.getState().createPod({
          runnerId: 1,
          agentTypeId: 2,
          repositoryId: 3,
          ticketId: 4,
          initialPrompt: "Hello",
          branchName: "feature/test",
        });
      });

      expect(podApi.create).toHaveBeenCalledWith({
        runner_id: 1,
        agent_type_id: 2,
        repository_id: 3,
        ticket_id: 4,
        initial_prompt: "Hello",
        branch_name: "feature/test",
      });
    });

    it("should use default agent_type_id when not provided", async () => {
      vi.mocked(podApi.create).mockResolvedValue({ pod: mockPod, message: "success" });

      await act(async () => {
        await usePodStore.getState().createPod({
          runnerId: 1,
        });
      });

      expect(podApi.create).toHaveBeenCalledWith(
        expect.objectContaining({
          agent_type_id: 0,
        })
      );
    });

    it("should add new pod to beginning of list", async () => {
      usePodStore.setState({ pods: [mockPod2] });
      vi.mocked(podApi.create).mockResolvedValue({ pod: mockPod, message: "success" });

      await act(async () => {
        await usePodStore.getState().createPod({ runnerId: 1 });
      });

      const state = usePodStore.getState();
      expect(state.pods[0]).toEqual(mockPod);
      expect(state.pods[1]).toEqual(mockPod2);
    });

    it("should handle create error", async () => {
      vi.mocked(podApi.create).mockRejectedValue({ message: "Create failed" });

      await expect(
        act(async () => {
          await usePodStore.getState().createPod({ runnerId: 1 });
        })
      ).rejects.toEqual({ message: "Create failed" });

      const state = usePodStore.getState();
      expect(state.error).toBe("Create failed");
    });
  });

  describe("terminatePod", () => {
    beforeEach(() => {
      usePodStore.setState({
        pods: [mockPod, mockPod2],
        currentPod: mockPod,
      });
    });

    it("should terminate pod successfully", async () => {
      vi.mocked(podApi.terminate).mockResolvedValue({ message: "success" });

      await act(async () => {
        await usePodStore.getState().terminatePod("pod-abc-123");
      });

      const state = usePodStore.getState();
      expect(state.pods[0].status).toBe("terminated");
      expect(state.currentPod?.status).toBe("terminated");
    });

    it("should only update the terminated pod", async () => {
      vi.mocked(podApi.terminate).mockResolvedValue({ message: "success" });

      await act(async () => {
        await usePodStore.getState().terminatePod("pod-abc-123");
      });

      const state = usePodStore.getState();
      expect(state.pods[1].status).toBe("running"); // mockPod2 unchanged
    });

    it("should not update currentPod if different key", async () => {
      vi.mocked(podApi.terminate).mockResolvedValue({ message: "success" });

      await act(async () => {
        await usePodStore.getState().terminatePod("pod-def-456");
      });

      const state = usePodStore.getState();
      expect(state.currentPod?.status).toBe("running"); // mockPod unchanged
    });

    it("should handle terminate error", async () => {
      const error = new MockApiError("Terminate failed", 500);
      vi.mocked(podApi.terminate).mockRejectedValue(error);

      await expect(
        act(async () => {
          await usePodStore.getState().terminatePod("pod-abc-123");
        })
      ).rejects.toThrow("Terminate failed");

      const state = usePodStore.getState();
      expect(state.error).toBe("Terminate failed");
    });

    it("should treat 404 as already terminated", async () => {
      const error = new MockApiError("Not found", 404);
      vi.mocked(podApi.terminate).mockRejectedValue(error);

      // Should not throw for 404
      await act(async () => {
        await usePodStore.getState().terminatePod("pod-abc-123");
      });

      const state = usePodStore.getState();
      // Pod should still be marked as terminated locally
      expect(state.pods[0].status).toBe("terminated");
      expect(state.error).toBeNull();
    });
  });

  describe("setCurrentPod", () => {
    it("should set current pod", () => {
      act(() => {
        usePodStore.getState().setCurrentPod(mockPod);
      });

      const state = usePodStore.getState();
      expect(state.currentPod).toEqual(mockPod);
    });

    it("should set to null", () => {
      usePodStore.setState({ currentPod: mockPod });

      act(() => {
        usePodStore.getState().setCurrentPod(null);
      });

      const state = usePodStore.getState();
      expect(state.currentPod).toBeNull();
    });
  });

  describe("updatePodStatus", () => {
    beforeEach(() => {
      usePodStore.setState({
        pods: [mockPod, mockPod2],
        currentPod: mockPod,
      });
    });

    it("should update pod status in list", () => {
      act(() => {
        usePodStore.getState().updatePodStatus("pod-abc-123", "paused");
      });

      const state = usePodStore.getState();
      expect(state.pods[0].status).toBe("paused");
    });

    it("should update currentPod status if matching", () => {
      act(() => {
        usePodStore.getState().updatePodStatus("pod-abc-123", "failed");
      });

      const state = usePodStore.getState();
      expect(state.currentPod?.status).toBe("failed");
    });

    it("should not update currentPod if different key", () => {
      act(() => {
        usePodStore.getState().updatePodStatus("pod-def-456", "paused");
      });

      const state = usePodStore.getState();
      expect(state.currentPod?.status).toBe("running");
    });

    it("should not affect other pods", () => {
      act(() => {
        usePodStore.getState().updatePodStatus("pod-abc-123", "terminated");
      });

      const state = usePodStore.getState();
      expect(state.pods[1].status).toBe("running");
    });

    it("should update error fields when provided", () => {
      act(() => {
        usePodStore
          .getState()
          .updatePodStatus(
            "pod-abc-123",
            "error",
            undefined,
            "GIT_AUTH_FAILED",
            "authentication failed for repository"
          );
      });

      const state = usePodStore.getState();
      expect(state.pods[0].status).toBe("error");
      expect(state.pods[0].error_code).toBe("GIT_AUTH_FAILED");
      expect(state.pods[0].error_message).toBe(
        "authentication failed for repository"
      );
    });

    it("should update error fields on currentPod when matching", () => {
      act(() => {
        usePodStore
          .getState()
          .updatePodStatus(
            "pod-abc-123",
            "error",
            undefined,
            "SANDBOX_FAILED",
            "sandbox creation failed"
          );
      });

      const state = usePodStore.getState();
      expect(state.currentPod?.status).toBe("error");
      expect(state.currentPod?.error_code).toBe("SANDBOX_FAILED");
      expect(state.currentPod?.error_message).toBe(
        "sandbox creation failed"
      );
    });

    it("should not set error fields when not provided", () => {
      act(() => {
        usePodStore.getState().updatePodStatus("pod-abc-123", "running");
      });

      const state = usePodStore.getState();
      expect(state.pods[0].status).toBe("running");
      expect(state.pods[0].error_code).toBeUndefined();
      expect(state.pods[0].error_message).toBeUndefined();
    });
  });

  describe("updateAgentStatus", () => {
    beforeEach(() => {
      usePodStore.setState({
        pods: [mockPod, mockPod2],
        currentPod: mockPod,
      });
    });

    it("should update agent status in list", () => {
      act(() => {
        usePodStore.getState().updateAgentStatus("pod-abc-123", "waiting");
      });

      const state = usePodStore.getState();
      expect(state.pods[0].agent_status).toBe("waiting");
    });

    it("should update currentPod agent status if matching", () => {
      act(() => {
        usePodStore.getState().updateAgentStatus("pod-abc-123", "idle");
      });

      const state = usePodStore.getState();
      expect(state.currentPod?.agent_status).toBe("idle");
    });

    it("should not update currentPod if different key", () => {
      act(() => {
        usePodStore.getState().updateAgentStatus("pod-def-456", "idle");
      });

      const state = usePodStore.getState();
      expect(state.currentPod?.agent_status).toBe("executing");
    });
  });

  describe("clearError", () => {
    it("should clear error", () => {
      usePodStore.setState({ error: "Some error" });

      act(() => {
        usePodStore.getState().clearError();
      });

      const state = usePodStore.getState();
      expect(state.error).toBeNull();
    });
  });
});
