import { request, orgPath } from "./base";

// Mesh types
export interface MeshNodeData {
  pod_key: string;
  status: string;
  agent_status: string;
  model?: string;
  ticket_id?: number;
  repository_id?: number;
  created_by_id: number;
  runner_id: number;
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

export interface MeshTopologyData {
  nodes: MeshNodeData[];
  edges: MeshEdgeData[];
  channels: ChannelInfoData[];
}

// Mesh API
export const meshApi = {
  // Get topology
  getTopology: () => {
    return request<{ topology: MeshTopologyData }>(orgPath("/mesh/topology"));
  },

  // For channel operations, use channelApi.joinPod() and channelApi.leavePod()
};
