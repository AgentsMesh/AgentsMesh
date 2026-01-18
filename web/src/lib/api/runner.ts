import { request, orgPath } from "./base";

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
    request<{ runner: RunnerData }>(`${orgPath("/runners")}/${id}`),

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
};
