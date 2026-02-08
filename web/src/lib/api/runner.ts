import { request, publicRequest, orgPath } from "./base";

// Runner interface matching the store
export interface RunnerData {
  id: number;
  node_id: string;
  description?: string;
  status: "online" | "offline" | "maintenance" | "busy";
  last_heartbeat?: string;
  current_pods: number;
  max_concurrent_pods: number;
  runner_version?: string;
  is_enabled: boolean;
  host_info?: {
    os?: string;
    arch?: string;
    memory?: number;
    cpu_cores?: number;
    hostname?: string;
  };
  // New field from Runner handshake - list of available agent type slugs
  available_agents?: string[];
  created_at: string;
  updated_at: string;
  active_pods?: Array<{
    pod_key: string;
    status: string;
    agent_status: string;
  }>;
}

// Relay connection info reported by Runner
// Note: session_id removed - channel routing now uses podKey directly
export interface RelayConnectionInfo {
  pod_key: string;
  relay_url: string;
  connected: boolean;
  connected_at: string;
}

// gRPC Registration Token interface
export interface GRPCRegistrationToken {
  id: number;
  organization_id: number;
  name?: string;
  labels?: string[];
  single_use: boolean;
  max_uses: number;
  used_count: number;
  expires_at: string;
  created_by?: number;
  created_at: string;
}

export const runnerApi = {
  list: (status?: string) => {
    const params = status ? `?status=${status}` : "";
    return request<{ runners: RunnerData[] }>(`${orgPath("/runners")}${params}`);
  },

  listAvailable: () =>
    request<{ runners: RunnerData[] }>(orgPath("/runners/available")),

  get: (id: number) =>
    request<{ runner: RunnerData; relay_connections?: RelayConnectionInfo[] }>(`${orgPath("/runners")}/${id}`),

  update: (id: number, data: { description?: string; max_concurrent_pods?: number; is_enabled?: boolean }) =>
    request<{ runner: RunnerData }>(`${orgPath("/runners")}/${id}`, {
      method: "PUT",
      body: data,
    }),

  delete: (id: number) =>
    request<{ message: string }>(`${orgPath("/runners")}/${id}`, {
      method: "DELETE",
    }),

  // gRPC Registration Token APIs (new unified system)
  createToken: (data?: { name?: string; labels?: string[]; max_uses?: number; expires_in_days?: number }) =>
    request<{ token: string; expires_at: string; message: string }>(orgPath("/runners/grpc/tokens"), {
      method: "POST",
      body: data || {},
    }),

  listTokens: () =>
    request<{ tokens: GRPCRegistrationToken[] }>(orgPath("/runners/grpc/tokens")),

  deleteToken: (id: number) =>
    request<{ message: string }>(`${orgPath("/runners/grpc/tokens")}/${id}`, {
      method: "DELETE",
    }),

  // Pod list for a specific runner
  listPods: (id: number, params?: { status?: string; limit?: number; offset?: number }) => {
    const searchParams = new URLSearchParams();
    if (params?.status) searchParams.set("status", params.status);
    if (params?.limit) searchParams.set("limit", params.limit.toString());
    if (params?.offset) searchParams.set("offset", params.offset.toString());
    const queryString = searchParams.toString();
    return request<{ pods: RunnerPodData[]; total: number; limit: number; offset: number }>(
      `${orgPath("/runners")}/${id}/pods${queryString ? `?${queryString}` : ""}`
    );
  },

  // Query sandbox status for specified pod keys
  querySandboxes: (id: number, podKeys: string[]) =>
    request<{ sandboxes: SandboxStatus[]; error?: string }>(`${orgPath("/runners")}/${id}/sandboxes/query`, {
      method: "POST",
      body: { pod_keys: podKeys },
    }),
};

// Pod data returned from Runner pods endpoint
export interface RunnerPodData {
  id: number;
  pod_key: string;
  organization_id: number;
  runner_id: number;
  agent_type_id?: number;
  custom_agent_type_id?: number;
  repository_id?: number;
  ticket_id?: number;
  status: string;
  claude_status?: string;
  branch_name?: string;
  sandbox_path?: string;
  session_id?: string;
  source_pod_key?: string;
  initial_prompt?: string;
  created_by_id: number;
  created_at: string;
  updated_at: string;
  terminated_at?: string;
}

// Sandbox status returned from sandbox query
export interface SandboxStatus {
  pod_key: string;
  exists: boolean;
  can_resume: boolean;
  sandbox_path?: string;
  repository_url?: string;
  branch_name?: string;
  current_commit?: string;
  size_bytes?: number;
  last_modified?: number;
  has_uncommitted_changes?: boolean;
  error?: string;
}

// Runner Authorization Status (for interactive registration)
export interface RunnerAuthStatus {
  status: "pending" | "authorized" | "expired";
  node_id?: string;
  expires_at?: string;
}

// Runner Authorization Response
export interface RunnerAuthorizeResponse {
  runner_id: number;
  node_id: string;
  message: string;
}

// Runner Authorization API (public, no auth required for status check)
export const runnerAuthApi = {
  // Get pending authorization status (public, no auth required)
  getAuthStatus: (authKey: string) =>
    publicRequest<RunnerAuthStatus>(`/api/v1/runners/grpc/auth-status?key=${authKey}`),

  // Authorize a Runner (requires auth, org-scoped)
  authorize: (orgSlug: string, authKey: string, nodeId?: string) =>
    request<RunnerAuthorizeResponse>(`/api/v1/orgs/${orgSlug}/runners/grpc/authorize`, {
      method: "POST",
      body: { auth_key: authKey, node_id: nodeId },
    }),
};
