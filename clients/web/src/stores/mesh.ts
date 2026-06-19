import { create } from "zustand";
import { useMemo } from "react";
import { fromBinary } from "@bufbuild/protobuf";
import { MeshTopologySchema } from "@proto/mesh/v1/mesh_pb";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";
import { getMeshService, getMeshState } from "@/lib/wasm-core";
import { topologyToCache } from "./meshTopologyToCache";
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

// Read side (B, zero-JSON): decode state proto bytes + topologyToCache once per
// _tick. The 6 per-node getters below + useTopology share this projection, so a
// render that queries several nodes never re-decodes the whole graph; the memo
// invalidates whenever _tick bumps (fetch / realtime patch).
let topoMemo: { tick: number; topo: MeshTopology | null } | null = null;
function readTopology(): MeshTopology | null {
  const tick = useMeshStore.getState()._tick;
  if (topoMemo?.tick === tick) return topoMemo.topo;
  const bytes = svc().topology_bytes();
  const topo = bytes.length === 0 ? null : topologyToCache(fromBinary(MeshTopologySchema, bytes));
  topoMemo = { tick, topo };
  return topo;
}

// Test-only: the memo keys on the monotonic _tick, which production never
// resets. Unit tests reinitialize the store to _tick=0 between cases, so they
// must drop the stale memo to re-read the freshly-mocked topology bytes.
export function __resetTopologyMemoForTests(): void {
  topoMemo = null;
}

export function useTopology(): MeshTopology | null {
  const tick = useMeshStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => readTopology(), [tick]);
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

  // Per-node queries derive from the single topology projection (zero-JSON).
  getNodeByKey: (podKey) => readTopology()?.nodes.find((n) => n.pod_key === podKey),
  getEdgesForNode: (podKey) =>
    readTopology()?.edges.filter((e) => e.source === podKey || e.target === podKey) ?? [],
  getChannelsForNode: (podKey) =>
    readTopology()?.channels.filter((c) => c.pod_keys.includes(podKey)) ?? [],
  getActiveNodes: () =>
    readTopology()?.nodes.filter((n) => n.status === "running" || n.status === "creating") ?? [],
  getNodesByRunner: (runnerId) =>
    readTopology()?.nodes.filter((n) => n.runner_id === runnerId) ?? [],
  getRunnerInfo: (runnerId) => readTopology()?.runners.find((r) => r.id === runnerId),
}));

reconnectRegistry.register({
  name: "mesh:topology",
  fn: () => useMeshStore.getState().fetchTopology?.(),
  priority: "deferred",
});
