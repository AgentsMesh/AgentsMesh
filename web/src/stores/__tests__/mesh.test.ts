import { describe, it, expect, beforeEach, vi } from "vitest";
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
    });
  });

  describe("fetchTopology", () => {
    it("should fetch topology successfully", async () => {
      vi.mocked(meshApi.getTopology).mockResolvedValue({
        topology: mockTopology,
      });

      await act(async () => {
        await useMeshStore.getState().fetchTopology();
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

      await act(async () => {
        await useMeshStore.getState().fetchTopology();
      });

      const state = useMeshStore.getState();
      expect(state.error).toBe("Network error");
      expect(state.loading).toBe(false);
    });

    it("should handle non-Error rejection", async () => {
      vi.mocked(meshApi.getTopology).mockRejectedValue("Unknown error");

      await act(async () => {
        await useMeshStore.getState().fetchTopology();
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

  describe("clearError", () => {
    it("should clear error", () => {
      useMeshStore.setState({ error: "Some error" });

      act(() => {
        useMeshStore.getState().clearError();
      });

      expect(useMeshStore.getState().error).toBeNull();
    });
  });

  describe("getNodeByKey", () => {
    beforeEach(() => {
      useMeshStore.setState({ topology: mockTopology });
    });

    it("should find node by key", () => {
      const node = useMeshStore.getState().getNodeByKey("pod-abc");
      expect(node).toEqual(mockNode1);
    });

    it("should return undefined for non-existent key", () => {
      const node = useMeshStore.getState().getNodeByKey("non-existent");
      expect(node).toBeUndefined();
    });

    it("should return undefined when topology is null", () => {
      useMeshStore.setState({ topology: null });
      const node = useMeshStore.getState().getNodeByKey("pod-abc");
      expect(node).toBeUndefined();
    });
  });

  describe("getEdgesForNode", () => {
    beforeEach(() => {
      useMeshStore.setState({ topology: mockTopology });
    });

    it("should find edges for source node", () => {
      const edges = useMeshStore.getState().getEdgesForNode("pod-abc");
      expect(edges).toHaveLength(1);
      expect(edges[0]).toEqual(mockEdge);
    });

    it("should find edges for target node", () => {
      const edges = useMeshStore.getState().getEdgesForNode("pod-def");
      expect(edges).toHaveLength(1);
      expect(edges[0]).toEqual(mockEdge);
    });

    it("should return empty array for node with no edges", () => {
      const edges = useMeshStore.getState().getEdgesForNode("pod-ghi");
      expect(edges).toEqual([]);
    });

    it("should return empty array when topology is null", () => {
      useMeshStore.setState({ topology: null });
      const edges = useMeshStore.getState().getEdgesForNode("pod-abc");
      expect(edges).toEqual([]);
    });
  });

  describe("getChannelsForNode", () => {
    beforeEach(() => {
      useMeshStore.setState({ topology: mockTopology });
    });

    it("should find channels for node", () => {
      const channels = useMeshStore.getState().getChannelsForNode("pod-abc");
      expect(channels).toHaveLength(1);
      expect(channels[0]).toEqual(mockChannel);
    });

    it("should return empty array for node with no channels", () => {
      const channels = useMeshStore.getState().getChannelsForNode("pod-ghi");
      expect(channels).toEqual([]);
    });

    it("should return empty array when topology is null", () => {
      useMeshStore.setState({ topology: null });
      const channels = useMeshStore.getState().getChannelsForNode("pod-abc");
      expect(channels).toEqual([]);
    });
  });

  describe("getActiveNodes", () => {
    beforeEach(() => {
      useMeshStore.setState({ topology: mockTopology });
    });

    it("should return only running and initializing nodes", () => {
      const activeNodes = useMeshStore.getState().getActiveNodes();
      expect(activeNodes).toHaveLength(2);
      expect(activeNodes.map((n) => n.pod_key)).toContain("pod-abc");
      expect(activeNodes.map((n) => n.pod_key)).toContain("pod-def");
      expect(activeNodes.map((n) => n.pod_key)).not.toContain("pod-ghi");
    });

    it("should include initializing nodes", () => {
      const initializingNode: MeshNode = {
        pod_key: "pod-init",
        status: "initializing",
        agent_status: "idle",
        model: "test",
        created_by_id: 1,
        runner_id: 4,
        runner_node_id: "runner-delta",
        runner_status: "online",
        started_at: "2024-01-01T00:00:00Z",
      };
      useMeshStore.setState({
        topology: {
          ...mockTopology,
          nodes: [...mockTopology.nodes, initializingNode],
        },
      });

      const activeNodes = useMeshStore.getState().getActiveNodes();
      expect(activeNodes.map((n) => n.pod_key)).toContain("pod-init");
    });

    it("should return empty array when topology is null", () => {
      useMeshStore.setState({ topology: null });
      const activeNodes = useMeshStore.getState().getActiveNodes();
      expect(activeNodes).toEqual([]);
    });
  });
});

describe("Helper Functions", () => {
  describe("getPodStatusInfo", () => {
    it("should return correct info for initializing status", () => {
      const info = getPodStatusInfo("initializing");
      expect(info.label).toBe("Initializing");
      expect(info.color).toBe("text-blue-600 dark:text-blue-400");
      expect(info.bgColor).toBe("bg-blue-100 dark:bg-blue-900/30");
    });

    it("should return correct info for running status", () => {
      const info = getPodStatusInfo("running");
      expect(info.label).toBe("Running");
      expect(info.color).toBe("text-green-600 dark:text-green-400");
    });

    it("should return correct info for paused status", () => {
      const info = getPodStatusInfo("paused");
      expect(info.label).toBe("Paused");
      expect(info.color).toBe("text-yellow-600 dark:text-yellow-400");
    });

    it("should return correct info for terminated status", () => {
      const info = getPodStatusInfo("terminated");
      expect(info.label).toBe("Terminated");
      expect(info.color).toBe("text-gray-600 dark:text-gray-400");
    });

    it("should return correct info for failed status", () => {
      const info = getPodStatusInfo("failed");
      expect(info.label).toBe("Failed");
      expect(info.color).toBe("text-red-600 dark:text-red-400");
    });

    it("should return terminated info for unknown status", () => {
      const info = getPodStatusInfo("unknown");
      expect(info).toEqual(getPodStatusInfo("terminated"));
    });
  });

  describe("getAgentStatusInfo", () => {
    it("should return correct info for executing status", () => {
      const info = getAgentStatusInfo("executing");
      expect(info.label).toBe("Executing");
      expect(info.color).toBe("text-green-600 dark:text-green-400");
      expect(info.dotColor).toBe("bg-green-500");
      expect(info.bgColor).toBe("bg-green-500/10");
      expect(info.icon).toBeDefined();
    });

    it("should return correct info for waiting status", () => {
      const info = getAgentStatusInfo("waiting");
      expect(info.label).toBe("Waiting for Input");
      expect(info.color).toBe("text-amber-600 dark:text-amber-400");
      expect(info.dotColor).toBe("bg-amber-500");
      expect(info.bgColor).toBe("bg-amber-500/10");
      expect(info.icon).toBeDefined();
    });

    it("should return correct info for idle status", () => {
      const info = getAgentStatusInfo("idle");
      expect(info.label).toBe("Idle");
      expect(info.color).toBe("text-gray-500 dark:text-gray-400");
      expect(info.dotColor).toBe("bg-gray-400");
      expect(info.bgColor).toBe("bg-gray-400/10");
      expect(info.icon).toBeDefined();
    });

    it("should return idle info as fallback for unknown status", () => {
      const info = getAgentStatusInfo("unknown-status");
      const idleInfo = getAgentStatusInfo("idle");
      expect(info).toEqual(idleInfo);
    });
  });

  describe("getBindingStatusInfo", () => {
    it("should return correct info for active status", () => {
      const info = getBindingStatusInfo("active");
      expect(info.label).toBe("Active");
      expect(info.color).toBe("stroke-green-500");
    });

    it("should return correct info for pending status", () => {
      const info = getBindingStatusInfo("pending");
      expect(info.label).toBe("Pending");
      expect(info.color).toBe("stroke-yellow-500");
    });

    it("should return correct info for revoked status", () => {
      const info = getBindingStatusInfo("revoked");
      expect(info.label).toBe("Revoked");
      expect(info.color).toBe("stroke-red-500");
    });

    it("should return correct info for expired status", () => {
      const info = getBindingStatusInfo("expired");
      expect(info.label).toBe("Expired");
      expect(info.color).toBe("stroke-gray-500");
    });

    it("should return active info for unknown status", () => {
      const info = getBindingStatusInfo("unknown");
      expect(info).toEqual(getBindingStatusInfo("active"));
    });
  });
});
