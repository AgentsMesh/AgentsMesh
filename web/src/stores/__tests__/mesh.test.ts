import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { act } from "@testing-library/react";
import {
  useMeshStore,
  getPodStatusInfo,
  getAgentStatusInfo,
  getBindingStatusInfo,
  MeshTopology,
  MeshNode,
  MeshEdge,
  ChannelInfo,
} from "../mesh";

// Mock the mesh API
vi.mock("@/lib/api", () => ({
  meshApi: {
    getTopology: vi.fn(),
  },
}));

import { meshApi } from "@/lib/api";

const mockNode1: MeshNode = {
  pod_key: "pod-abc",
  status: "running",
  agent_status: "executing",
  model: "claude-code",
  created_by_id: 1,
  runner_id: 1,
  runner_node_id: "runner-alpha",
  runner_status: "online",
  started_at: "2024-01-01T00:00:00Z",
};

const mockNode2: MeshNode = {
  pod_key: "pod-def",
  status: "running",
  agent_status: "waiting",
  model: "gpt-engineer",
  created_by_id: 1,
  runner_id: 2,
  runner_node_id: "runner-beta",
  runner_status: "online",
  started_at: "2024-01-02T00:00:00Z",
};

const mockNode3: MeshNode = {
  pod_key: "pod-ghi",
  status: "terminated",
  agent_status: "idle",
  model: "claude-code",
  created_by_id: 1,
  runner_id: 3,
  runner_node_id: "runner-gamma",
  runner_status: "offline",
  started_at: "2024-01-03T00:00:00Z",
};

const mockEdge: MeshEdge = {
  id: 1,
  source: "pod-abc",
  target: "pod-def",
  granted_scopes: ["read", "write"],
  status: "active",
};

const mockChannel: ChannelInfo = {
  id: 1,
  name: "general",
  pod_keys: ["pod-abc", "pod-def"],
  message_count: 10,
  is_archived: false,
};

const mockTopology: MeshTopology = {
  nodes: [mockNode1, mockNode2, mockNode3],
  edges: [mockEdge],
  channels: [mockChannel],
};

describe("Mesh Store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset store to initial state
    useMeshStore.setState({
      topology: null,
      selectedNode: null,
      selectedChannel: null,
      loading: false,
      error: null,
      nodePositions: {},
    });
  });

  describe("initial state", () => {
    it("should have default values", () => {
      const state = useMeshStore.getState();

      expect(state.topology).toBeNull();
      expect(state.selectedNode).toBeNull();
      expect(state.selectedChannel).toBeNull();
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
      expect(state.nodePositions).toEqual({});
    });
  });

  describe("fetchTopology", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("should fetch topology successfully", async () => {
      vi.mocked(meshApi.getTopology).mockResolvedValue({
        topology: mockTopology,
      });

      act(() => {
        useMeshStore.getState().fetchTopology();
      });
      // Advance past the 500ms debounce window and flush the async work
      await act(async () => {
        vi.advanceTimersByTime(500);
      });

      const state = useMeshStore.getState();
      expect(state.topology).toEqual(mockTopology);
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
    });

    it("should handle fetch error", async () => {
      vi.mocked(meshApi.getTopology).mockRejectedValue(
        new Error("Network error")
      );

      act(() => {
        useMeshStore.getState().fetchTopology();
      });
      await act(async () => {
        vi.advanceTimersByTime(500);
      });

      const state = useMeshStore.getState();
      expect(state.error).toBe("Network error");
      expect(state.loading).toBe(false);
    });

    it("should handle non-Error rejection", async () => {
      vi.mocked(meshApi.getTopology).mockRejectedValue("Unknown error");

      act(() => {
        useMeshStore.getState().fetchTopology();
      });
      await act(async () => {
        vi.advanceTimersByTime(500);
      });

      const state = useMeshStore.getState();
      expect(state.error).toBe("Failed to fetch topology");
    });
  });

  describe("selectNode", () => {
    it("should select a node", () => {
      act(() => {
        useMeshStore.getState().selectNode("pod-abc");
      });

      const state = useMeshStore.getState();
      expect(state.selectedNode).toBe("pod-abc");
    });

    it("should clear selectedChannel when selecting node", () => {
      useMeshStore.setState({ selectedChannel: 1 });

      act(() => {
        useMeshStore.getState().selectNode("pod-abc");
      });

      const state = useMeshStore.getState();
      expect(state.selectedNode).toBe("pod-abc");
      expect(state.selectedChannel).toBeNull();
    });

    it("should set to null", () => {
      useMeshStore.setState({ selectedNode: "pod-abc" });

      act(() => {
        useMeshStore.getState().selectNode(null);
      });

      const state = useMeshStore.getState();
      expect(state.selectedNode).toBeNull();
    });
  });

  describe("selectChannel", () => {
    it("should select a channel", () => {
      act(() => {
        useMeshStore.getState().selectChannel(1);
      });

      const state = useMeshStore.getState();
      expect(state.selectedChannel).toBe(1);
    });

    it("should clear selectedNode when selecting channel", () => {
      useMeshStore.setState({ selectedNode: "pod-abc" });

      act(() => {
        useMeshStore.getState().selectChannel(1);
      });

      const state = useMeshStore.getState();
      expect(state.selectedChannel).toBe(1);
      expect(state.selectedNode).toBeNull();
    });

    it("should set to null", () => {
      useMeshStore.setState({ selectedChannel: 1 });

      act(() => {
        useMeshStore.getState().selectChannel(null);
      });

      const state = useMeshStore.getState();
      expect(state.selectedChannel).toBeNull();
    });
  });

  // Note: Polling has been removed - realtime events handle updates now

  describe("updateNodePosition", () => {
    it("should save position for a node", () => {
      act(() => {
        useMeshStore.getState().updateNodePosition("runner-group-1", { x: 100, y: 200 });
      });

      const state = useMeshStore.getState();
      expect(state.nodePositions["runner-group-1"]).toEqual({ x: 100, y: 200 });
    });

    it("should update position for an existing node", () => {
      useMeshStore.setState({
        nodePositions: { "runner-group-1": { x: 50, y: 50 } },
      });

      act(() => {
        useMeshStore.getState().updateNodePosition("runner-group-1", { x: 300, y: 400 });
      });

      const state = useMeshStore.getState();
      expect(state.nodePositions["runner-group-1"]).toEqual({ x: 300, y: 400 });
    });

    it("should preserve positions of other nodes", () => {
      useMeshStore.setState({
        nodePositions: { "runner-group-1": { x: 10, y: 20 } },
      });

      act(() => {
        useMeshStore.getState().updateNodePosition("runner-group-2", { x: 500, y: 0 });
      });

      const state = useMeshStore.getState();
      expect(state.nodePositions["runner-group-1"]).toEqual({ x: 10, y: 20 });
      expect(state.nodePositions["runner-group-2"]).toEqual({ x: 500, y: 0 });
    });
  });

  describe("clearError", () => {
    it("should clear error", () => {
      useMeshStore.setState({ error: "Some error" });

      act(() => {
        useMeshStore.getState().clearError();
      });

      expect(useMeshStore.getState().error).toBeNull();
    });
  });

});
