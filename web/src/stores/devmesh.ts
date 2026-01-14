import { create } from "zustand";
import { devmeshApi, DevMeshNodeData, DevMeshEdgeData, ChannelInfoData, DevMeshTopologyData } from "@/lib/api/client";
import { getErrorMessage } from "@/lib/utils";

// Re-export API types for use in components
export type DevMeshNode = DevMeshNodeData;
export type DevMeshEdge = DevMeshEdgeData;
export type ChannelInfo = ChannelInfoData;
export type DevMeshTopology = DevMeshTopologyData;

// Request to create a pod for a ticket
export interface CreatePodForTicketRequest {
  runner_id: number;
  initial_prompt?: string;
  model?: string;
  permission_mode?: string;
  think_level?: string;
}

interface DevMeshState {
  // State
  topology: DevMeshTopology | null;
  selectedNode: string | null;
  selectedChannel: number | null;
  loading: boolean;
  error: string | null;

  // Actions
  fetchTopology: () => Promise<void>;
  selectNode: (podKey: string | null) => void;
  selectChannel: (channelId: number | null) => void;
  clearError: () => void;

  // Node helpers
  getNodeByKey: (podKey: string) => DevMeshNode | undefined;
  getEdgesForNode: (podKey: string) => DevMeshEdge[];
  getChannelsForNode: (podKey: string) => ChannelInfo[];
  getActiveNodes: () => DevMeshNode[];
}

export const useDevMeshStore = create<DevMeshState>((set, get) => ({
  topology: null,
  selectedNode: null,
  selectedChannel: null,
  loading: false,
  error: null,

  fetchTopology: async () => {
    set({ loading: true, error: null });
    try {
      const response = await devmeshApi.getTopology();
      set({ topology: response.topology, loading: false });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch topology"),
        loading: false,
      });
    }
  },

  selectNode: (podKey) => {
    set({ selectedNode: podKey, selectedChannel: null });
  },

  selectChannel: (channelId) => {
    set({ selectedChannel: channelId, selectedNode: null });
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
}));

// Helper function to get pod status display info
export const getPodStatusInfo = (status: string) => {
  const statusMap: Record<
    string,
    { label: string; color: string; bgColor: string }
  > = {
    initializing: {
      label: "Initializing",
      color: "text-blue-600",
      bgColor: "bg-blue-100",
    },
    running: {
      label: "Running",
      color: "text-green-600",
      bgColor: "bg-green-100",
    },
    paused: {
      label: "Paused",
      color: "text-yellow-600",
      bgColor: "bg-yellow-100",
    },
    terminated: {
      label: "Terminated",
      color: "text-gray-600",
      bgColor: "bg-gray-100",
    },
    failed: {
      label: "Failed",
      color: "text-red-600",
      bgColor: "bg-red-100",
    },
  };
  return statusMap[status] || statusMap.terminated;
};

// Helper function to get agent status display info
export const getAgentStatusInfo = (agentStatus: string) => {
  const statusMap: Record<
    string,
    { label: string; color: string; icon: string }
  > = {
    idle: { label: "Idle", color: "text-gray-500", icon: "⏸" },
    thinking: { label: "Thinking", color: "text-blue-500", icon: "🤔" },
    coding: { label: "Coding", color: "text-green-500", icon: "💻" },
    testing: { label: "Testing", color: "text-yellow-500", icon: "🧪" },
    reviewing: { label: "Reviewing", color: "text-purple-500", icon: "📝" },
    waiting: { label: "Waiting", color: "text-orange-500", icon: "⏳" },
    error: { label: "Error", color: "text-red-500", icon: "❌" },
  };
  return statusMap[agentStatus] || { label: agentStatus, color: "text-gray-500", icon: "•" };
};

// Helper function to get binding status display info
export const getBindingStatusInfo = (status: string) => {
  const statusMap: Record<
    string,
    { label: string; color: string }
  > = {
    active: { label: "Active", color: "stroke-green-500" },
    pending: { label: "Pending", color: "stroke-yellow-500" },
    revoked: { label: "Revoked", color: "stroke-red-500" },
    expired: { label: "Expired", color: "stroke-gray-500" },
  };
  return statusMap[status] || statusMap.active;
};
