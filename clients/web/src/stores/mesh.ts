import { create } from "zustand";
import { useMemo } from "react";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";
import { getMeshService, getMeshState } from "@/lib/wasm-core";
import { useIDEStore } from "./ide";
import { useChannelStore } from "./channel";

export interface MeshNode {
  pod_key: string; alias?: string; status: string;
  agent_status?: string; agent_slug?: string; runner_id?: number;
  model?: string; title?: string; ticket_id?: number; ticket_slug?: string;
  ticket_title?: string; repository_id?: number; created_by_id?: number;
  runner_node_id?: string; runner_status?: string; started_at?: string;
}
export interface MeshEdge {
  id?: number; source: string; target: string;
  binding_status?: string; status?: string;
  granted_scopes?: string[]; pending_scopes?: string[];
}
export interface ChannelInfo {
  id: number; name: string; description?: string;
  pod_keys: string[]; message_count?: number; is_archived?: boolean;
}
export interface RunnerInfo {
  id: number; name: string; status: string;
  node_id?: string; max_concurrent_pods?: number; current_pods?: number;
  pod_keys?: string[];
}
export interface MeshTopology {
  nodes: MeshNode[]; edges: MeshEdge[];
  channels: ChannelInfo[]; runners: RunnerInfo[];
}

export { getPodStatusInfo, getAgentStatusInfo, getBindingStatusInfo } from "./meshHelpers";

export interface CreatePodForTicketRequest {
  runner_id: number;
  prompt?: string;
  model?: string;
  permission_mode?: string;
}

// runtime.state.mesh (getMeshState) is the read/write SSOT; getMeshService is
// networking-only. Node status/agent patch in via Rust event_dispatch (see
// realtimePodHandlers); structure (nodes/edges/channels) still needs a full fetch.
const svc = getMeshState;
const net = getMeshService;
const bump = () => useMeshStore.setState((s) => ({ _tick: s._tick + 1 }));

export function useTopology(): MeshTopology | null {
  const tick = useMeshStore((s) => s._tick);
  return useMemo(() => {
    const raw = svc().topology_json();
    return raw ? (JSON.parse(typeof raw === "string" ? raw : JSON.stringify(raw)) as MeshTopology) : null;
  }, [tick]);
}

interface MeshState {
  _tick: number;
  selectedNode: string | null;
  selectedChannel: number | null;
  loading: boolean;
  error: string | null;
  nodePositions: Record<string, { x: number; y: number }>;

  fetchTopology: () => void;
  cancelPendingTopologyFetch: () => void;
  selectNode: (podKey: string | null) => void;
  selectChannel: (channelId: number | null) => void;
  updateNodePosition: (nodeId: string, position: { x: number; y: number }) => void;
  clearError: () => void;

  getNodeByKey: (podKey: string) => MeshNode | undefined;
  getEdgesForNode: (podKey: string) => MeshEdge[];
  getChannelsForNode: (podKey: string) => ChannelInfo[];
  getActiveNodes: () => MeshNode[];
  getNodesByRunner: (runnerId: number) => MeshNode[];
  getRunnerInfo: (runnerId: number) => RunnerInfo | undefined;
}

let topologyDebounceTimer: ReturnType<typeof setTimeout> | null = null;

export const useMeshStore = create<MeshState>((set, get) => ({
  _tick: 0,
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
        const bytes = await net().fetch_topology();
        svc().replace_topology(bytes);
        set({ loading: false, _tick: get()._tick + 1 });
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
    svc().select_node(podKey ?? undefined);
    const raw = svc().selected_node();
    set({ selectedNode: raw ? String(raw) : null, selectedChannel: null });
  },

  selectChannel: (channelId) => {
    if (channelId !== null) {
      useIDEStore.getState().setActiveActivity("channels");
      useChannelStore.getState().setSelectedChannelId(channelId);
    }
    set({ selectedChannel: channelId, selectedNode: null });
  },

  updateNodePosition: (nodeId, position) => {
    set((state) => ({ nodePositions: { ...state.nodePositions, [nodeId]: position } }));
  },

  clearError: () => set({ error: null }),

  getNodeByKey: (podKey) => {
    const raw = svc().get_node_json(podKey);
    return raw ? JSON.parse(String(raw)) : undefined;
  },
  getEdgesForNode: (podKey) => JSON.parse(svc().get_edges_for_node_json(podKey)),
  getChannelsForNode: (podKey) => JSON.parse(svc().get_channels_for_node_json(podKey)),
  getActiveNodes: () => JSON.parse(svc().get_active_nodes_json()),
  getNodesByRunner: (runnerId) => JSON.parse(svc().get_nodes_by_runner_json(BigInt(runnerId))),
  getRunnerInfo: (runnerId) => {
    const raw = svc().get_runner_info_json(BigInt(runnerId));
    return raw ? JSON.parse(String(raw)) : undefined;
  },
}));

reconnectRegistry.register({
  name: "mesh:topology",
  fn: () => useMeshStore.getState().fetchTopology?.(),
  priority: "deferred",
});
