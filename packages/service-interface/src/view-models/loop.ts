// Loop UI ViewModels — snake_case shapes for loop form / store / kanban.
// Kept snake_case because Rust SSOT (state/loop_state.rs) serializes
// LoopData with serde snake_case JSON for set_loops round-trip. Owned here
// (zero-dep contract layer) so the web fromProtoLoop projection and the
// desktop electron-adapter projection share one type definition.

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
  /**
   * Ordered list of EnvBundle names attached to every run. Each name is
   * emitted as a `USE_ENV_BUNDLE "<name>"` line in the generated AgentFile
   * (in array order; later entries override earlier ones on conflicting
   * env keys). Empty/absent = no bundle.
   */
  used_env_bundles: string[];
  config_overrides?: Record<string, unknown>;
  execution_mode: ExecutionMode;
  cron_expression?: string;
  callback_url?: string;
  autopilot_config: Record<string, unknown>;
  status: LoopStatus;
  sandbox_strategy: SandboxStrategy;
  session_persistence: boolean;
  concurrency_policy: ConcurrencyPolicy;
  max_concurrent_runs: number;
  max_retained_runs: number;
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
  trigger_type: string;
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
  used_env_bundles?: string[];
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
  used_env_bundles?: string[];
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
