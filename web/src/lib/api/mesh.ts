import { request, orgPath } from "./base";

// Mesh types
export interface MeshNodeData {
  pod_key: string;
  status: string;
  agent_status: string;
  model?: string;
  title?: string; // OSC 0/2 terminal title
  ticket_id?: number;
  repository_id?: number;
  created_by_id: number;
  runner_id: number;
  runner_node_id: string;
  runner_status: string;
  ticket_identifier?: string;
  ticket_title?: string;
  started_at?: string;
  position?: { x: number; y: number };
}

export interface MeshEdgeData {
  id: number;
  source: string;
  target: string;
  granted_scopes: string[];
  pending_scopes?: string[];
  status: string;
}

export interface ChannelInfoData {
  id: number;
  name: string;
  description?: string;
  pod_keys: string[];
  message_count: number;
  is_archived: boolean;
}

export interface RunnerInfoData {
  id: number;
  node_id: string;
  status: string;
  max_concurrent_pods: number;
  current_pods: number;
}

export interface MeshTopologyData {
  nodes: MeshNodeData[];
  edges: MeshEdgeData[];
  channels: ChannelInfoData[];
  runners?: RunnerInfoData[];
}

// Mesh API
export const meshApi = {
  // Get topology
  getTopology: () => {
    return request<{ topology: MeshTopologyData }>(orgPath("/mesh/topology"));
  },

  // For channel operations, use channelApi.joinPod() and channelApi.leavePod()
};
