import { create } from "zustand";
import { meshApi, MeshNodeData, MeshEdgeData, ChannelInfoData, MeshTopologyData, RunnerInfoData } from "@/lib/api";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";
import { useIDEStore } from "./ide";
import { useChannelStore } from "./channel";

// Re-export API types with cleaner names for component use
export type MeshNode = MeshNodeData;
export type MeshEdge = MeshEdgeData;
export type ChannelInfo = ChannelInfoData;
export type MeshTopology = MeshTopologyData;
export type RunnerInfo = RunnerInfoData;

// Request to create a pod for a ticket
export interface CreatePodForTicketRequest {
  runner_id: number;
  prompt?: string;
  model?: string;
  permission_mode?: string;
}

interface MeshState {
  // State
  topology: MeshTopology | null;
  selectedNode: string | null;
  selectedChannel: number | null;
  loading: boolean;
  error: string | null;
  nodePositions: Record<string, { x: number; y: number }>;

  // Actions
  fetchTopology: () => void;
  cancelPendingTopologyFetch: () => void;
  selectNode: (podKey: string | null) => void;
  selectChannel: (channelId: number | null) => void;
  updateNodePosition: (nodeId: string, position: { x: number; y: number }) => void;
  clearError: () => void;

  // Node helpers
  getNodeByKey: (podKey: string) => MeshNode | undefined;
  getEdgesForNode: (podKey: string) => MeshEdge[];
  getChannelsForNode: (podKey: string) => ChannelInfo[];
  getActiveNodes: () => MeshNode[];
  getNodesByRunner: (runnerId: number) => MeshNode[];
  getRunnerInfo: (runnerId: number) => RunnerInfo | undefined;
}

// Debounce timer for fetchTopology — coalesce rapid pod events into a single API call.
let topologyDebounceTimer: ReturnType<typeof setTimeout> | null = null;

export const useMeshStore = create<MeshState>((set, get) => ({
  topology: null,
  selectedNode: null,
  selectedChannel: null,
  loading: false,
  error: null,
  nodePositions: {},

  fetchTopology: () => {
    if (topologyDebounceTimer) clearTimeout(topologyDebounceTimer);
    topologyDebounceTimer = setTimeout(async () => {
      topologyDebounceTimer = null;
      set({ loading: true, error: null });
      try {
        const response = await meshApi.getTopology();
        set({ topology: response.topology, loading: false });
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to fetch topology"), loading: false });
      }
    }, 500);
  },

  cancelPendingTopologyFetch: () => {
    if (topologyDebounceTimer) {
      clearTimeout(topologyDebounceTimer);
      topologyDebounceTimer = null;
    }
  },

  selectNode: (podKey) => {
    set({ selectedNode: podKey, selectedChannel: null });
  },

  selectChannel: (channelId) => {
    if (channelId !== null) {
      // Navigate to Channels tab and select the channel
      useIDEStore.getState().setActiveActivity("channels");
      useChannelStore.getState().setSelectedChannelId(channelId);
    }
    set({ selectedChannel: channelId, selectedNode: null });
  },

  updateNodePosition: (nodeId, position) => {
    set((state) => ({
      nodePositions: {
        ...state.nodePositions,
        [nodeId]: position,
      },
    }));
  },

  clearError: () => {
    set({ error: null });
  },

  getNodeByKey: (podKey) => {
    const { topology } = get();
    return topology?.nodes.find((n) => n.pod_key === podKey);
  },

  getEdgesForNode: (podKey) => {
    const { topology } = get();
    if (!topology) return [];
    return topology.edges.filter(
      (e) => e.source === podKey || e.target === podKey
    );
  },

  getChannelsForNode: (podKey) => {
    const { topology } = get();
    if (!topology) return [];
    return topology.channels.filter((c) =>
      c.pod_keys.includes(podKey)
    );
  },

  getActiveNodes: () => {
    const { topology } = get();
    if (!topology) return [];
    return topology.nodes.filter(
      (n) => n.status === "running" || n.status === "initializing"
    );
  },

  getNodesByRunner: (runnerId) => {
    const { topology } = get();
    if (!topology) return [];
    return topology.nodes.filter((n) => n.runner_id === runnerId);
  },

  getRunnerInfo: (runnerId) => {
    const { topology } = get();
    return topology?.runners?.find((r) => r.id === runnerId);
  },
}));

export { getPodStatusInfo, getAgentStatusInfo, getBindingStatusInfo } from "./mesh-status-info";

reconnectRegistry.register({
  name: "mesh:topology",
  fn: () => useMeshStore.getState().fetchTopology?.(),
  priority: "deferred",
});
