export interface AutopilotStatusChangedData {
  autopilot_controller_key: string;
  pod_key: string;
  phase: string;
  current_iteration: number;
  max_iterations: number;
  circuit_breaker_state: string;
  circuit_breaker_reason?: string;
}

export interface AutopilotIterationData {
  autopilot_controller_key: string;
  iteration: number;
  phase: string;
  summary?: string;
  files_changed?: string[];
  duration_ms?: number;
}

export interface AutopilotCreatedData {
  autopilot_controller_key: string;
  pod_key: string;
}

export interface AutopilotTerminatedData {
  autopilot_controller_key: string;
  reason: string;
}

export interface AutopilotThinkingData {
  autopilot_controller_key: string;
  iteration: number;
  decision_type:
    | "continue" | "completed" | "need_help" | "give_up"
    | "CONTINUE" | "TASK_COMPLETED" | "NEED_HUMAN_HELP" | "GIVE_UP";
  reasoning: string;
  confidence: number;
  action?: {
    type: "observe" | "send_input" | "wait" | "none";
    content: string;
    reason: string;
  };
  progress?: {
    summary: string;
    completed_steps: string[];
    remaining_steps: string[];
    percent: number;
  };
  help_request?: {
    reason: string;
    context: string;
    terminal_excerpt: string;
    suggestions: Array<{
      action: string;
      label: string;
    }>;
  };
}

export interface MREventData {
  mr_id: number;
  mr_iid: number;
  mr_url: string;
  source_branch: string;
  target_branch?: string;
  title?: string;
  state: string;
  action?: string; // opened, updated, merged, closed
  ticket_id?: number;
  ticket_slug?: string;
  pod_id?: number;
  repository_id: number;
  pipeline_status?: string;
}

export interface PipelineEventData {
  mr_id?: number;
  pipeline_id: number;
  pipeline_status: string;
  pipeline_url?: string;
  source_branch?: string;
  ticket_id?: number;
  ticket_slug?: string;
  pod_id?: number;
  repository_id: number;
}

export interface LoopRunEventData {
  loop_id: number;
  run_id: number;
  run_number: number;
  status: string;
  pod_key?: string;
}

export interface NotificationPayloadData {
  source: string;
  title: string;
  body: string;
  link?: string;
  priority: "normal" | "high";
  channels: Record<string, boolean>;
}

export interface LoopRunWarningData {
  loop_id: number;
  run_id: number;
  run_number: number;
  warning: string;
  detail?: string;
}
