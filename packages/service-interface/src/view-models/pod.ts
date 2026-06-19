// Pod ViewModels — UI-side projection of `proto.pod.v1.Pod`. Owned in this
// zero-dep contract layer so web (fromProtoPod) and desktop (electron-adapter
// projection) share one definition.

// Interaction mode — mirrors agentfile.ModePTY / agentfile.ModeACP. The
// POD_MODE_PTY / POD_MODE_ACP value constants stay in clients/web's
// lib/pod-modes.ts (which now re-exports this type).
export type PodMode = "pty" | "acp";

export interface PodData {
  id?: number;
  pod_key: string;
  status: "initializing" | "running" | "paused" | "disconnected" | "orphaned" | "completed" | "terminated" | "error" | "failed";
  agent_status?: string;
  prompt?: string;
  branch_name?: string;
  sandbox_path?: string;
  started_at?: string;
  finished_at?: string;
  last_activity?: string;
  created_at?: string;
  title?: string;
  alias?: string;
  runner?: { id?: number; node_id?: string; status?: string };
  agent?: { name?: string; slug?: string };
  repository?: { id?: number; name?: string; slug?: string; provider_type?: string };
  ticket?: { id?: number; slug?: string; title?: string };
  loop?: { id?: number; name?: string; slug?: string };
  interaction_mode?: PodMode;
  perpetual?: boolean;
  restart_count?: number;
  last_restart_at?: string;
  error_code?: string;
  error_message?: string;
  created_by?: { id?: number; username?: string; name?: string };
  // Query-derived (ListPods only): the active pod resumed from this one.
  resumed_by_pod_key?: string;
}
