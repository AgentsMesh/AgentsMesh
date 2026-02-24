import { create } from "zustand";
import { meshApi, MeshNodeData, MeshEdgeData, ChannelInfoData, MeshTopologyData, RunnerInfoData } from "@/lib/api";
import { getErrorMessage } from "@/lib/utils";
import { Play, Hourglass, Pause, type LucideIcon } from "lucide-react";

// Re-export API types with cleaner names for component use
export type MeshNode = MeshNodeData;
export type MeshEdge = MeshEdgeData;
export type ChannelInfo = ChannelInfoData;
export type MeshTopology = MeshTopologyData;
export type RunnerInfo = RunnerInfoData;

// Request to create a pod for a ticket
export interface CreatePodForTicketRequest {
  runner_id: number;
  initial_prompt?: string;
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
  nodePositions: Record<string, { x: number; y: number }>; // Cached drag positions for Runner Group nodes

  // Actions
  fetchTopology: () => Promise<void>;
  selectNode: (podKey: string | null) => void;
  selectChannel: (channelId: number | null) => void;
  updateNodeTitle: (podKey: string, title: string) => void;
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

export const useMeshStore = create<MeshState>((set, get) => ({
  topology: null,
  selectedNode: null,
  selectedChannel: null,
  loading: false,
  error: null,
  nodePositions: {},

  fetchTopology: async () => {
    set({ loading: true, error: null });
    try {
      const response = await meshApi.getTopology();
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

  updateNodeTitle: (podKey, title) => {
    set((state) => ({
      topology: state.topology
        ? {
            ...state.topology,
            nodes: state.topology.nodes.map((n) =>
              n.pod_key === podKey ? { ...n, title } : n
            ),
          }
        : null,
    }));
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

// Helper function to get pod status display info
export const getPodStatusInfo = (status: string) => {
  const statusMap: Record<
    string,
    { label: string; color: string; bgColor: string }
  > = {
    initializing: {
      label: "Initializing",
      color: "text-blue-600 dark:text-blue-400",
      bgColor: "bg-blue-100 dark:bg-blue-900/30",
    },
    running: {
      label: "Running",
      color: "text-green-600 dark:text-green-400",
      bgColor: "bg-green-100 dark:bg-green-900/30",
    },
    paused: {
      label: "Paused",
      color: "text-yellow-600 dark:text-yellow-400",
      bgColor: "bg-yellow-100 dark:bg-yellow-900/30",
    },
    terminated: {
      label: "Terminated",
      color: "text-gray-600 dark:text-gray-400",
      bgColor: "bg-gray-100 dark:bg-gray-800",
    },
    failed: {
      label: "Failed",
      color: "text-red-600 dark:text-red-400",
      bgColor: "bg-red-100 dark:bg-red-900/30",
    },
  };
  return statusMap[status] || statusMap.terminated;
};

// Helper function to get agent status display info
export const getAgentStatusInfo = (agentStatus: string): {
  label: string;
  color: string;
  dotColor: string;
  bgColor: string;
  icon: LucideIcon;
} => {
  const statusMap: Record<string, {
    label: string;
    color: string;
    dotColor: string;
    bgColor: string;
    icon: LucideIcon;
  }> = {
    executing: {
      label: "Executing",
      color: "text-green-600 dark:text-green-400",
      dotColor: "bg-green-500",
      bgColor: "bg-green-500/10",
      icon: Play,
    },
    waiting: {
      label: "Waiting for Input",
      color: "text-amber-600 dark:text-amber-400",
      dotColor: "bg-amber-500",
      bgColor: "bg-amber-500/10",
      icon: Hourglass,
    },
    idle: {
      label: "Idle",
      color: "text-gray-500 dark:text-gray-400",
      dotColor: "bg-gray-400",
      bgColor: "bg-gray-400/10",
      icon: Pause,
    },
  };
  return statusMap[agentStatus] || statusMap.idle;
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
