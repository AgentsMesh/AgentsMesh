import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { act } from "@testing-library/react";

const mockFetchTopology = vi.fn();
const mockReplaceTopology = vi.fn();
const mockSelectNode = vi.fn();
const mockSelectedNode = vi.fn();
// Read side (B): the per-node queries derive from topology_bytes in TS.
const mockTopologyBytes = vi.fn(() => new Uint8Array());

const noopSvc = new Proxy({}, {
  get: () => () => "[]",
});

vi.mock("@/lib/wasm-core", () => ({
  // networking-only after the SSOT migration
  getMeshService: () => ({
    fetch_topology: mockFetchTopology,
  }),
  // runtime.state.mesh surface: reads + select + replace
  getMeshState: () => ({
    replace_topology: mockReplaceTopology,
    select_node: mockSelectNode,
    selected_node: mockSelectedNode,
    topology_bytes: mockTopologyBytes,
  }),
  getChannelService: () => noopSvc,
  getChannelState: () => noopSvc,
}));

import { create, toBinary } from "@bufbuild/protobuf";
import { MeshTopologySchema } from "@proto/mesh/v1/mesh_pb";
import {
  useMeshStore,
  __resetTopologyMemoForTests,
  MeshTopology,
  MeshNode,
  MeshEdge,
  ChannelInfo,
} from "../mesh";

// Encode a view topology → proto bytes so mockTopologyBytes feeds the same
// fromBinary + topologyToCache path the store uses (B read side).
function setTopo(t: MeshTopology) {
  const proto = create(MeshTopologySchema, {
    nodes: t.nodes.map((n) => ({
      podKey: n.pod_key, status: n.status, agentStatus: n.agent_status ?? "",
      agentSlug: n.agent_slug ?? "", runnerId: BigInt(n.runner_id ?? 0),
    })),
    edges: t.edges.map((e) => ({ source: e.source, target: e.target, status: e.status ?? "" })),
    channels: t.channels.map((c) => ({ id: BigInt(c.id), name: c.name, podKeys: c.pod_keys })),
    runners: t.runners.map((r) => ({ id: BigInt(r.id), status: r.status, nodeId: r.node_id ?? "" })),
  });
  mockTopologyBytes.mockReturnValue(toBinary(MeshTopologySchema, proto));
}

const mockNode1: MeshNode = {
  pod_key: "pod-abc",
  status: "running",
  agent_status: "executing",
  agent_slug: "claude-code",
  runner_id: 1,
};

const mockNode2: MeshNode = {
  pod_key: "pod-def",
  status: "running",
  agent_status: "waiting",
  agent_slug: "gpt-engineer",
  runner_id: 2,
};

const mockNode3: MeshNode = {
  pod_key: "pod-ghi",
  status: "terminated",
  agent_status: "idle",
  agent_slug: "claude-code",
  runner_id: 3,
};

const mockEdge: MeshEdge = {
  source: "pod-abc",
  target: "pod-def",
  binding_status: "active",
};

const mockChannel: ChannelInfo = {
  id: 1,
  name: "general",
  pod_keys: ["pod-abc", "pod-def"],
};

const mockTopology: MeshTopology = {
  nodes: [mockNode1, mockNode2, mockNode3],
  edges: [mockEdge],
  channels: [mockChannel],
  runners: [],
};

describe("Mesh Store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    __resetTopologyMemoForTests();
    useMeshStore.setState({
      _tick: 0,
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
      mockFetchTopology.mockResolvedValue(undefined);

      act(() => {
        useMeshStore.getState().fetchTopology();
      });
      await act(async () => {
        vi.advanceTimersByTime(500);
      });

      const state = useMeshStore.getState();
      expect(mockFetchTopology).toHaveBeenCalled();
      expect(state.loading).toBe(false);
      expect(state.error).toBeNull();
    });

    it("should handle fetch error", async () => {
      mockFetchTopology.mockRejectedValue(new Error("Network error"));

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
      mockFetchTopology.mockRejectedValue("Unknown error");

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
      mockSelectedNode.mockReturnValue("pod-abc");

      act(() => {
        useMeshStore.getState().selectNode("pod-abc");
      });

      expect(mockSelectNode).toHaveBeenCalledWith("pod-abc");
      const state = useMeshStore.getState();
      expect(state.selectedNode).toBe("pod-abc");
    });

    it("should clear selectedChannel when selecting node", () => {
      useMeshStore.setState({ selectedChannel: 1 });
      mockSelectedNode.mockReturnValue("pod-abc");

      act(() => {
        useMeshStore.getState().selectNode("pod-abc");
      });

      const state = useMeshStore.getState();
      expect(state.selectedNode).toBe("pod-abc");
      expect(state.selectedChannel).toBeNull();
    });

    it("should set to null", () => {
      useMeshStore.setState({ selectedNode: "pod-abc" });
      mockSelectedNode.mockReturnValue(null);

      act(() => {
        useMeshStore.getState().selectNode(null);
      });

      expect(mockSelectNode).toHaveBeenCalledWith(undefined);
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

  // Read side (B): helpers derive from topology_bytes + topologyToCache, so
  // assertions match key fields (the proto round-trip adds defaulted fields).
  describe("WASM state reader helpers", () => {
    it("getNodeByKey returns the matching node", () => {
      setTopo(mockTopology);
      const result = useMeshStore.getState().getNodeByKey("pod-abc");
      expect(result).toMatchObject({ pod_key: "pod-abc", agent_slug: "claude-code", runner_id: 1 });
    });

    it("getNodeByKey returns undefined for missing node", () => {
      setTopo(mockTopology);
      expect(useMeshStore.getState().getNodeByKey("missing")).toBeUndefined();
    });

    it("getNodeByKey returns undefined when no topology", () => {
      mockTopologyBytes.mockReturnValue(new Uint8Array());
      expect(useMeshStore.getState().getNodeByKey("pod-abc")).toBeUndefined();
    });

    it("getEdgesForNode filters edges touching the node", () => {
      setTopo(mockTopology);
      const result = useMeshStore.getState().getEdgesForNode("pod-abc");
      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({ source: "pod-abc", target: "pod-def" });
    });

    it("getChannelsForNode filters channels containing the node", () => {
      setTopo(mockTopology);
      const result = useMeshStore.getState().getChannelsForNode("pod-abc");
      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({ id: 1, name: "general" });
    });

    it("getActiveNodes returns running/creating nodes only", () => {
      setTopo(mockTopology);
      const result = useMeshStore.getState().getActiveNodes();
      // mockNode1/2 are running, mockNode3 is terminated.
      expect(result.map((n) => n.pod_key)).toEqual(["pod-abc", "pod-def"]);
    });

    it("getNodesByRunner filters by runner id", () => {
      setTopo(mockTopology);
      const result = useMeshStore.getState().getNodesByRunner(1);
      expect(result.map((n) => n.pod_key)).toEqual(["pod-abc"]);
    });

    it("getRunnerInfo returns the matching runner", () => {
      setTopo({ ...mockTopology, runners: [{ id: 5, name: "", status: "online" }] });
      const result = useMeshStore.getState().getRunnerInfo(5);
      expect(result).toMatchObject({ id: 5, status: "online" });
    });

    it("getRunnerInfo returns undefined for missing runner", () => {
      setTopo(mockTopology);
      expect(useMeshStore.getState().getRunnerInfo(999)).toBeUndefined();
    });
  });
});
