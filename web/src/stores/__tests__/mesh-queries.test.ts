import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import {
  useMeshStore,
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

describe("Mesh Store - Query Helpers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useMeshStore.setState({
      topology: null,
      selectedNode: null,
      selectedChannel: null,
      loading: false,
      error: null,
      nodePositions: {},
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
