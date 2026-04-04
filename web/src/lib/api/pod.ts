import { request, orgPath } from "./base";
import type { PodMode } from "@/lib/pod-modes";

// Pod interface matching the store
export interface PodData {
  id: number;
  pod_key: string;
  status: "initializing" | "running" | "paused" | "disconnected" | "orphaned" | "completed" | "terminated" | "error" | "failed";
  agent_status: string;
  initial_prompt?: string;
  branch_name?: string;
  sandbox_path?: string;
  started_at?: string;
  finished_at?: string;
  last_activity?: string;
  created_at: string;
  title?: string; // OSC 0/2 terminal title
  alias?: string; // User-defined display name
  runner?: {
    id: number;
    node_id: string;
    status: string;
  };
  agent?: {
    name: string;
    slug: string;
  };
  repository?: {
    id: number;
    name: string;
    slug: string;
    provider_type?: string; // github, gitlab, gitee
  };
  ticket?: {
    id: number;
    slug: string;
    title: string;
  };
  loop?: {
    id: number;
    name: string;
    slug: string;
  };
  interaction_mode?: PodMode;
  error_code?: string;
  error_message?: string;
  created_by?: {
    id: number;
    username: string;
    name?: string;
  };
}

// Pods API
export const podApi = {
  list: (filters?: { status?: string; runnerId?: number; createdById?: number; limit?: number; offset?: number }) => {
    const params = new URLSearchParams();
    if (filters?.status) params.append("status", filters.status);
    if (filters?.runnerId) params.append("runner_id", String(filters.runnerId));
    if (filters?.createdById) params.append("created_by_id", String(filters.createdById));
    if (filters?.limit) params.append("limit", String(filters.limit));
    if (filters?.offset) params.append("offset", String(filters.offset));
    const query = params.toString() ? `?${params.toString()}` : "";
    return request<{ pods: PodData[]; total: number; limit: number; offset: number }>(`${orgPath("/pods")}${query}`);
  },

  get: (key: string) =>
    request<{ pod: PodData }>(`${orgPath("/pods")}/${key}`),

  create: (data: {
    agent_slug?: string; // Required unless resuming
    runner_id?: number;
    ticket_slug?: string;
    alias?: string; // User-defined display name (max 100 chars)
    cols?: number; // Terminal columns (from xterm.js)
    rows?: number; // Terminal rows (from xterm.js)
    // AgentFile Layer — SSOT (PROMPT, MODE, CONFIG, REPO, BRANCH, CREDENTIAL)
    agentfile_layer?: string;
    // Resume mode fields
    source_pod_key?: string;
    resume_agent_session?: boolean;
  }) =>
    request<{ message: string; pod: PodData }>(
      orgPath("/pods"),
      {
        method: "POST",
        body: data,
      }
    ),

  terminate: (key: string) =>
    request<{ message: string }>(`${orgPath("/pods")}/${key}/terminate`, {
      method: "POST",
    }),

  // Get connection info for WebSocket terminal
  getConnectionInfo: (key: string) =>
    request<{ pod_key: string; ws_url: string; status: string }>(
      `${orgPath("/pods")}/${key}/connect`
    ),

  // Get Pod connection info via Relay
  // Returns Relay URL and token for WebSocket connection
  // Note: podKey is embedded in the token for channel routing
  getPodConnection: (key: string) =>
    request<{
      relay_url: string;
      token: string;
      pod_key: string;
    }>(`${orgPath("/pods")}/${key}/relay/connect`),

  // Update pod alias (user-defined display name)
  // Pass null to clear the alias
  updateAlias: (key: string, alias: string | null) =>
    request<{ message: string }>(`${orgPath("/pods")}/${key}/alias`, {
      method: "PATCH",
      body: { alias },
    }),
};
