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

  regenerateAuthToken: (id: number) =>
    request<{ auth_token: string; message: string }>(`${orgPath("/runners")}/${id}/regenerate-token`, {
      method: "POST",
    }),

  // Create one-time registration token
  createToken: () =>
    request<{ token: string; message: string }>(orgPath("/runners/tokens"), {
      method: "POST",
      body: JSON.stringify({}),
    }),
};
