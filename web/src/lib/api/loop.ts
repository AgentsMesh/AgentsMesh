import { request, orgPath } from "./base";

// Loop types
export type LoopStatus = "enabled" | "disabled" | "archived";
export type ExecutionMode = "autopilot" | "direct";
export type SandboxStrategy = "persistent" | "fresh";
export type ConcurrencyPolicy = "skip" | "queue" | "replace";
export type RunStatus = "pending" | "running" | "completed" | "failed" | "timeout" | "cancelled" | "skipped";

export interface LoopData {
  id: number;
  organization_id: number;
  name: string;
  slug: string;
  description?: string;
  agent_slug?: string;
  custom_agent_slug?: string;
  permission_mode: string;
  prompt_template: string;
  prompt_variables?: Record<string, unknown>;
  repository_id?: number;
  runner_id?: number;
  branch_name?: string;
  ticket_id?: number;
  credential_profile_id?: number;
  config_overrides?: Record<string, unknown>;
  execution_mode: ExecutionMode;
  cron_expression?: string; // If set, cron scheduling is enabled; manual/API trigger is always available
  callback_url?: string; // Webhook callback URL (POST run result on completion)
  autopilot_config: Record<string, unknown>;
  status: LoopStatus;
  sandbox_strategy: SandboxStrategy; // persistent = reuse sandbox + session; fresh = clean slate
  session_persistence: boolean; // true = keep agent session (conversation history) across runs
  concurrency_policy: ConcurrencyPolicy;
  max_concurrent_runs: number;
  max_retained_runs: number; // 0 = unlimited
  timeout_minutes: number;
  sandbox_path?: string;
  last_pod_key?: string;
  created_by_id: number;
  total_runs: number;
  successful_runs: number;
  failed_runs: number;
  active_run_count: number;
  avg_duration_sec?: number;
  last_run_at?: string;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

export interface LoopRunData {
  id: number;
  organization_id: number;
  loop_id: number;
  run_number: number;
  status: RunStatus;
  pod_key?: string;
  autopilot_controller_key?: string;
  trigger_type: string; // "cron" | "api" | "manual"
  trigger_source?: string;
  resolved_prompt?: string;
  started_at?: string;
  finished_at?: string;
  duration_sec?: number;
  exit_summary?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateLoopRequest {
  name: string;
  slug?: string;
  description?: string;
  agent_slug?: string;
  custom_agent_slug?: string;
  permission_mode?: string;
  prompt_template: string;
  prompt_variables?: Record<string, unknown>;
  repository_id?: number;
  runner_id?: number;
  branch_name?: string;
  ticket_id?: number;
  credential_profile_id?: number;
  config_overrides?: Record<string, unknown>;
  execution_mode?: string;
  cron_expression?: string;
  autopilot_config?: Record<string, unknown>;
  callback_url?: string;
  sandbox_strategy?: string;
  session_persistence?: boolean;
  concurrency_policy?: string;
  max_concurrent_runs?: number;
  max_retained_runs?: number;
  timeout_minutes?: number;
}

export interface UpdateLoopRequest {
  name?: string;
  description?: string;
  agent_slug?: string;
  custom_agent_slug?: string;
  permission_mode?: string;
  prompt_template?: string;
  prompt_variables?: Record<string, unknown>;
  repository_id?: number;
  runner_id?: number;
  branch_name?: string;
  ticket_id?: number;
  credential_profile_id?: number;
  config_overrides?: Record<string, unknown>;
  execution_mode?: string;
  cron_expression?: string;
  autopilot_config?: Record<string, unknown>;
  callback_url?: string;
  sandbox_strategy?: string;
  session_persistence?: boolean;
  concurrency_policy?: string;
  max_concurrent_runs?: number;
  max_retained_runs?: number;
  timeout_minutes?: number;
}

export const loopApi = {
  list: (filters?: {
    status?: string;
    execution_mode?: string;
    cron_enabled?: boolean;
    query?: string;
    limit?: number;
    offset?: number;
  }) => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== "") {
          params.set(key, String(value));
        }
      });
    }
    const qs = params.toString();
    return request<{ loops: LoopData[]; total: number; limit: number; offset: number }>(
      orgPath(`/loops${qs ? `?${qs}` : ""}`)
    );
  },

  get: (slug: string) =>
    request<{ loop: LoopData }>(orgPath(`/loops/${slug}`)),

  create: (data: CreateLoopRequest) =>
    request<{ loop: LoopData }>(orgPath("/loops"), {
      method: "POST",
      body: data,
    }),

  update: (slug: string, data: UpdateLoopRequest) =>
    request<{ loop: LoopData }>(orgPath(`/loops/${slug}`), {
      method: "PUT",
      body: data,
    }),

  delete: (slug: string) =>
    request<{ message: string }>(orgPath(`/loops/${slug}`), { method: "DELETE" }),

  enable: (slug: string) =>
    request<{ loop: LoopData }>(orgPath(`/loops/${slug}/enable`), { method: "POST" }),

  disable: (slug: string) =>
    request<{ loop: LoopData }>(orgPath(`/loops/${slug}/disable`), { method: "POST" }),

  trigger: (slug: string) =>
    request<{ run: LoopRunData; skipped?: boolean; reason?: string }>(
      orgPath(`/loops/${slug}/trigger`),
      { method: "POST" }
    ),

  listRuns: (slug: string, filters?: { status?: string; limit?: number; offset?: number }) => {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== "") {
          params.set(key, String(value));
        }
      });
    }
    const qs = params.toString();
    return request<{ runs: LoopRunData[]; total: number; limit: number; offset: number }>(
      orgPath(`/loops/${slug}/runs${qs ? `?${qs}` : ""}`)
    );
  },

  getRun: (slug: string, runId: number) =>
    request<{ run: LoopRunData }>(orgPath(`/loops/${slug}/runs/${runId}`)),

  cancelRun: (slug: string, runId: number) =>
    request<{ message: string }>(orgPath(`/loops/${slug}/runs/${runId}/cancel`), { method: "POST" }),
};
