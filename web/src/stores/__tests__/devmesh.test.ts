import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { act } from "@testing-library/react";
import {
  useDevMeshStore,
  getPodStatusInfo,
  getAgentStatusInfo,
  getBindingStatusInfo,
  DevMeshTopology,
  DevMeshNode,
  DevMeshEdge,
  ChannelInfo,
} from "../devmesh";

// Mock the devmesh API
vi.mock("@/lib/api/client", () => ({
  devmeshApi: {
    getTopology: vi.fn(),
  },
}));

import { devmeshApi } from "@/lib/api/client";

const mockNode1: DevMeshNode = {
  pod_key: "pod-abc",
  status: "running",
  agent_status: "coding",
  model: "claude-code",
  created_by_id: 1,
  runner_id: 1,
  started_at: "2024-01-01T00:00:00Z",
};

const mockNode2: DevMeshNode = {
  pod_key: "pod-def",
  status: "running",
  agent_status: "thinking",
  model: "gpt-engineer",
  created_by_id: 1,
  runner_id: 2,
  started_at: "2024-01-02T00:00:00Z",
};

const mockNode3: DevMeshNode = {
  pod_key: "pod-ghi",
  status: "terminated",
  agent_status: "idle",
  model: "claude-code",
  created_by_id: 1,
  runner_id: 3,
  started_at: "2024-01-03T00:00:00Z",
};

const mockEdge: DevMeshEdge = {
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

const mockTopology: DevMeshTopology = {
  nodes: [mockNode1, mockNode2, mockNode3],
  edges: [mockEdge],
  channels: [mockChannel],
};

describe("DevMesh Store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    // Reset store to initial state
    useDevMeshStore.setState({
      topology: null,
      selectedNode: null,
      selectedChannel: null,
      loading: false,
      error: null,
      pollInterval: null,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    // Clean up any polling intervals
    const state = useDevMeshStore.getState();
    if (state.pollInterval !== null) {
      clearInterval(state.pollInterval);
    }
  });

  describe("initial state", () => {
    it("should have default values", () => {
      const state = useDevMeshStore.getState();

      expect(state.topology).toBeNull();
      expect(state.selectedNode).toBeNull();
      expect(state.selectedChannel).toBeNull();
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
      expect(state.pollInterval).toBeNull();
    });
  });

  describe("fetchTopology", () => {
    it("should fetch topology successfully", async () => {
      vi.mocked(devmeshApi.getTopology).mockResolvedValue({
        topology: mockTopology,
      });

      await act(async () => {
        await useDevMeshStore.getState().fetchTopology();
      });

      const state = useDevMeshStore.getState();
      expect(state.topology).toEqual(mockTopology);
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
    });

    it("should handle fetch error", async () => {
      vi.mocked(devmeshApi.getTopology).mockRejectedValue(
        new Error("Network error")
      );

      await act(async () => {
        await useDevMeshStore.getState().fetchTopology();
      });

      const state = useDevMeshStore.getState();
      expect(state.error).toBe("Network error");
      expect(state.loading).toBe(false);
    });

    it("should handle non-Error rejection", async () => {
      vi.mocked(devmeshApi.getTopology).mockRejectedValue("Unknown error");

      await act(async () => {
        await useDevMeshStore.getState().fetchTopology();
      });

      const state = useDevMeshStore.getState();
      expect(state.error).toBe("Failed to fetch topology");
    });
  });

  describe("selectNode", () => {
    it("should select a node", () => {
      act(() => {
        useDevMeshStore.getState().selectNode("pod-abc");
      });

      const state = useDevMeshStore.getState();
      expect(state.selectedNode).toBe("pod-abc");
    });

    it("should clear selectedChannel when selecting node", () => {
      useDevMeshStore.setState({ selectedChannel: 1 });

      act(() => {
        useDevMeshStore.getState().selectNode("pod-abc");
      });

      const state = useDevMeshStore.getState();
      expect(state.selectedNode).toBe("pod-abc");
      expect(state.selectedChannel).toBeNull();
    });

    it("should set to null", () => {
      useDevMeshStore.setState({ selectedNode: "pod-abc" });

      act(() => {
        useDevMeshStore.getState().selectNode(null);
      });

      const state = useDevMeshStore.getState();
      expect(state.selectedNode).toBeNull();
    });
  });

  describe("selectChannel", () => {
    it("should select a channel", () => {
      act(() => {
        useDevMeshStore.getState().selectChannel(1);
      });

      const state = useDevMeshStore.getState();
      expect(state.selectedChannel).toBe(1);
    });

    it("should clear selectedNode when selecting channel", () => {
      useDevMeshStore.setState({ selectedNode: "pod-abc" });

      act(() => {
        useDevMeshStore.getState().selectChannel(1);
      });

      const state = useDevMeshStore.getState();
      expect(state.selectedChannel).toBe(1);
      expect(state.selectedNode).toBeNull();
    });

    it("should set to null", () => {
      useDevMeshStore.setState({ selectedChannel: 1 });

      act(() => {
        useDevMeshStore.getState().selectChannel(null);
      });

      const state = useDevMeshStore.getState();
      expect(state.selectedChannel).toBeNull();
    });
  });

  // Note: Polling has been removed - realtime events handle updates now

  describe("clearError", () => {
    it("should clear error", () => {
      useDevMeshStore.setState({ error: "Some error" });

      act(() => {
        useDevMeshStore.getState().clearError();
      });

      expect(useDevMeshStore.getState().error).toBeNull();
    });
  });

  describe("getNodeByKey", () => {
    beforeEach(() => {
      useDevMeshStore.setState({ topology: mockTopology });
    });

    it("should find node by key", () => {
      const node = useDevMeshStore.getState().getNodeByKey("pod-abc");
      expect(node).toEqual(mockNode1);
    });

    it("should return undefined for non-existent key", () => {
      const node = useDevMeshStore.getState().getNodeByKey("non-existent");
      expect(node).toBeUndefined();
    });

    it("should return undefined when topology is null", () => {
      useDevMeshStore.setState({ topology: null });
      const node = useDevMeshStore.getState().getNodeByKey("pod-abc");
      expect(node).toBeUndefined();
    });
  });

  describe("getEdgesForNode", () => {
    beforeEach(() => {
      useDevMeshStore.setState({ topology: mockTopology });
    });

    it("should find edges for source node", () => {
      const edges = useDevMeshStore.getState().getEdgesForNode("pod-abc");
      expect(edges).toHaveLength(1);
      expect(edges[0]).toEqual(mockEdge);
    });

    it("should find edges for target node", () => {
      const edges = useDevMeshStore.getState().getEdgesForNode("pod-def");
      expect(edges).toHaveLength(1);
      expect(edges[0]).toEqual(mockEdge);
    });

    it("should return empty array for node with no edges", () => {
      const edges = useDevMeshStore.getState().getEdgesForNode("pod-ghi");
      expect(edges).toEqual([]);
    });

    it("should return empty array when topology is null", () => {
      useDevMeshStore.setState({ topology: null });
      const edges = useDevMeshStore.getState().getEdgesForNode("pod-abc");
      expect(edges).toEqual([]);
    });
  });

  describe("getChannelsForNode", () => {
    beforeEach(() => {
      useDevMeshStore.setState({ topology: mockTopology });
    });

    it("should find channels for node", () => {
      const channels = useDevMeshStore.getState().getChannelsForNode("pod-abc");
      expect(channels).toHaveLength(1);
      expect(channels[0]).toEqual(mockChannel);
    });

    it("should return empty array for node with no channels", () => {
      const channels = useDevMeshStore.getState().getChannelsForNode("pod-ghi");
      expect(channels).toEqual([]);
    });

    it("should return empty array when topology is null", () => {
      useDevMeshStore.setState({ topology: null });
      const channels = useDevMeshStore.getState().getChannelsForNode("pod-abc");
      expect(channels).toEqual([]);
    });
  });

  describe("getActiveNodes", () => {
    beforeEach(() => {
      useDevMeshStore.setState({ topology: mockTopology });
    });

    it("should return only running and initializing nodes", () => {
      const activeNodes = useDevMeshStore.getState().getActiveNodes();
      expect(activeNodes).toHaveLength(2);
      expect(activeNodes.map((n) => n.pod_key)).toContain("pod-abc");
      expect(activeNodes.map((n) => n.pod_key)).toContain("pod-def");
      expect(activeNodes.map((n) => n.pod_key)).not.toContain("pod-ghi");
    });

    it("should include initializing nodes", () => {
      const initializingNode: DevMeshNode = {
        pod_key: "pod-init",
        status: "initializing",
        agent_status: "idle",
        model: "test",
        created_by_id: 1,
        runner_id: 4,
        started_at: "2024-01-01T00:00:00Z",
      };
      useDevMeshStore.setState({
        topology: {
          ...mockTopology,
          nodes: [...mockTopology.nodes, initializingNode],
        },
      });

      const activeNodes = useDevMeshStore.getState().getActiveNodes();
      expect(activeNodes.map((n) => n.pod_key)).toContain("pod-init");
    });

    it("should return empty array when topology is null", () => {
      useDevMeshStore.setState({ topology: null });
      const activeNodes = useDevMeshStore.getState().getActiveNodes();
      expect(activeNodes).toEqual([]);
    });
  });
});

describe("Helper Functions", () => {
  describe("getPodStatusInfo", () => {
    it("should return correct info for initializing status", () => {
      const info = getPodStatusInfo("initializing");
      expect(info.label).toBe("Initializing");
      expect(info.color).toBe("text-blue-600");
      expect(info.bgColor).toBe("bg-blue-100");
    });

    it("should return correct info for running status", () => {
      const info = getPodStatusInfo("running");
      expect(info.label).toBe("Running");
      expect(info.color).toBe("text-green-600");
    });

    it("should return correct info for paused status", () => {
      const info = getPodStatusInfo("paused");
      expect(info.label).toBe("Paused");
      expect(info.color).toBe("text-yellow-600");
    });

    it("should return correct info for terminated status", () => {
      const info = getPodStatusInfo("terminated");
      expect(info.label).toBe("Terminated");
      expect(info.color).toBe("text-gray-600");
    });

    it("should return correct info for failed status", () => {
      const info = getPodStatusInfo("failed");
      expect(info.label).toBe("Failed");
      expect(info.color).toBe("text-red-600");
    });

    it("should return terminated info for unknown status", () => {
      const info = getPodStatusInfo("unknown");
      expect(info).toEqual(getPodStatusInfo("terminated"));
    });
  });

  describe("getAgentStatusInfo", () => {
    it("should return correct info for idle status", () => {
      const info = getAgentStatusInfo("idle");
      expect(info.label).toBe("Idle");
      expect(info.color).toBe("text-gray-500");
      expect(info.icon).toBe("⏸");
    });

    it("should return correct info for thinking status", () => {
      const info = getAgentStatusInfo("thinking");
      expect(info.label).toBe("Thinking");
      expect(info.color).toBe("text-blue-500");
      expect(info.icon).toBe("🤔");
    });

    it("should return correct info for coding status", () => {
      const info = getAgentStatusInfo("coding");
      expect(info.label).toBe("Coding");
      expect(info.color).toBe("text-green-500");
      expect(info.icon).toBe("💻");
    });

    it("should return correct info for testing status", () => {
      const info = getAgentStatusInfo("testing");
      expect(info.label).toBe("Testing");
      expect(info.icon).toBe("🧪");
    });

    it("should return correct info for reviewing status", () => {
      const info = getAgentStatusInfo("reviewing");
      expect(info.label).toBe("Reviewing");
      expect(info.icon).toBe("📝");
    });

    it("should return correct info for waiting status", () => {
      const info = getAgentStatusInfo("waiting");
      expect(info.label).toBe("Waiting");
      expect(info.icon).toBe("⏳");
    });

    it("should return correct info for error status", () => {
      const info = getAgentStatusInfo("error");
      expect(info.label).toBe("Error");
      expect(info.color).toBe("text-red-500");
      expect(info.icon).toBe("❌");
    });

    it("should return fallback for unknown status", () => {
      const info = getAgentStatusInfo("unknown-status");
      expect(info.label).toBe("unknown-status");
      expect(info.color).toBe("text-gray-500");
      expect(info.icon).toBe("•");
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
