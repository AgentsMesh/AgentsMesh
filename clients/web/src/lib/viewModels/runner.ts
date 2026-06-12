/**
 * Runner ViewModels — UI-side projection of `proto.runner_api.v1` types.
 *
 * Snake_case fields are kept because (a) the wire format serializes
 * host_info as a JSON string the adapter parses, (b) RunnerPodData spans
 * the mesh plane that isn't covered by proto.runner_api.v1, and (c) too
 * many consumers (RunnerCard, RunnerLogsCard, useRunnerDetail) reference
 * these fields directly. Adapters in `lib/api/runnerConnect.ts` map the
 * camelCase proto types onto these ViewModels.
 */
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
  visibility?: "organization" | "private";
  registered_by_user_id?: number;
  host_info?: {
    os?: string;
    arch?: string;
    memory?: number;
    cpu_cores?: number;
    hostname?: string;
  };
  available_agents?: string[];
  tags?: string[];
  created_at?: string;
  updated_at?: string;
  active_pods?: Array<{
    pod_key: string;
    status: string;
    agent_status: string;
  }>;
}

export interface RelayConnectionInfo {
  pod_key: string;
  relay_url: string;
  connected: boolean;
  connected_at: string;
}

export interface GRPCRegistrationToken {
  id: number;
  organization_id?: number;
  name?: string;
  labels?: string[];
  single_use?: boolean;
  max_uses?: number;
  used_count?: number;
  expires_at?: string;
  created_by?: number;
  created_at?: string;
}

export interface RunnerListResponse {
  runners: RunnerData[];
  latest_runner_version?: string;
}

export interface RunnerDetailResponse {
  runner: RunnerData;
  relay_connections?: RelayConnectionInfo[];
  latest_runner_version?: string;
}

export interface RunnerLogData {
  id: number;
  request_id: string;
  status: string;
  storage_key?: string;
  size_bytes: number;
  error_message?: string;
  requested_by_id: number;
  download_url?: string;
  created_at?: string;
  completed_at?: string;
}

export interface RunnerPodData {
  id: number;
  pod_key: string;
  organization_id: number;
  runner_id: number;
  agent_slug?: string;
  custom_agent_slug?: string;
  repository_id?: number;
  ticket_id?: number;
  ticket?: {
    id: number;
    slug: string;
    title: string;
  };
  status: string;
  agent_status: string;
  claude_status?: string;
  branch_name?: string;
  sandbox_path?: string;
  session_id?: string;
  source_pod_key?: string;
  resumed_by_pod_key?: string;
  prompt?: string;
  created_by_id?: number;
  created_at?: string;
  updated_at?: string;
  terminated_at?: string;
}

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

export interface RunnerAuthStatus {
  status: "pending" | "authorized" | "expired";
  node_id?: string;
  expires_at?: string;
}

export interface RunnerAuthorizeResponse {
  runner_id: number;
  node_id: string;
  message: string;
}
