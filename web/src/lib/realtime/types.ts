/**
 * Event types from the backend
 */
export type EventType =
  // Entity events (broadcast to organization)
  | "pod:created"
  | "pod:status_changed"
  | "pod:agent_status_changed"
  | "pod:terminated"
  | "pod:title_changed"
  | "pod:init_progress"
  | "channel:message"
  | "channel:message_edited"
  | "channel:message_deleted"
  | "ticket:created"
  | "ticket:updated"
  | "ticket:status_changed"
  | "ticket:moved"
  | "ticket:deleted"
  | "runner:online"
  | "runner:offline"
  | "runner:updated"
  // AutopilotController events
  | "autopilot:status_changed"
  | "autopilot:iteration"
  | "autopilot:created"
  | "autopilot:terminated"
  | "autopilot:thinking"
  // MergeRequest events
  | "mr:created"
  | "mr:updated"
  | "mr:merged"
  | "mr:closed"
  // Pipeline events
  | "pipeline:updated"
  // Loop events
  | "loop_run:started"
  | "loop_run:completed"
  | "loop_run:failed"
  | "loop_run:warning"
  // Notification events (targeted to specific users)
  | "notification"
  | "terminal:notification"
  | "task:completed"
  | "mention:notification"
  // System events
  | "system:maintenance"
  // Connection events (client-side only)
  | "connected"
  | "ping"
  | "pong";

/**
 * Event categories
 */
export type EventCategory = "entity" | "notification" | "system";

/**
 * Base event structure from the server
 */
export interface RealtimeEvent<T = unknown> {
  type: EventType;
  category: EventCategory;
  organization_id: number;
  target_user_id?: number;
  target_user_ids?: number[];
  entity_type?: string;
  entity_id?: string;
  data: T;
  timestamp: number;
}

/**
 * Pod status changed event payload
 */
export interface PodStatusChangedData {
  pod_key: string;
  status: string;
  previous_status?: string;
  agent_status?: string;
  error_code?: string;
  error_message?: string;
}

/**
 * Pod created event payload
 */
export interface PodCreatedData {
  pod_key: string;
  status: string;
  agent_status?: string;
  runner_id: number;
  ticket_id?: number;
  ticket_slug?: string;
  created_by_id: number;
}

/**
 * Runner status event payload
 */
export interface RunnerStatusData {
  runner_id: number;
  node_id: string;
  status: string;
  current_pods?: number;
  last_heartbeat?: string;
}

/**
 * Ticket status changed event payload
 */
export interface TicketStatusChangedData {
  slug: string;
  status: string;
  previous_status?: string;
}

/**
 * Terminal notification event payload (OSC 777)
 */
export interface TerminalNotificationData {
  pod_key: string;
  title: string;
  body: string;
}

/**
 * Task completed event payload
 */
export interface TaskCompletedData {
  pod_key: string;
  agent_status: string;
  ticket_id?: number;
  ticket_slug?: string;
}

/**
 * Pod title changed event payload (OSC 0/2)
 */
export interface PodTitleChangedData {
  pod_key: string;
  title: string;
}

/**
 * Pod initialization progress event payload
 */
export interface PodInitProgressData {
  pod_key: string;
  phase: string; // pending, cloning, preparing, starting_pty, ready
  progress: number; // 0-100
  message: string; // Human-readable progress message
}

/**
 * Channel message event payload
 */
export interface ChannelMessageData {
  id: number;
  channel_id: number;
  sender_pod?: string;
  sender_user_id?: number;
  sender_name?: string;
  message_type: string;
  content: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

/**
 * Channel message edited event payload
 */
export interface ChannelMessageEditedData {
  id: number;
  channel_id: number;
  content: string;
  edited_at: string;
}

/**
 * Channel message deleted event payload
 */
export interface ChannelMessageDeletedData {
  id: number;
  channel_id: number;
}

/**
 * AutopilotController status changed event payload
 */
export interface AutopilotStatusChangedData {
  autopilot_controller_key: string;
  pod_key: string;
  phase: string;
  current_iteration: number;
  max_iterations: number;
  circuit_breaker_state: string;
  circuit_breaker_reason?: string;
}

/**
 * AutopilotController iteration event payload
 */
export interface AutopilotIterationData {
  autopilot_controller_key: string;
  iteration: number;
  phase: string;
  summary?: string;
  files_changed?: string[];
  duration_ms?: number;
}

/**
 * AutopilotController created event payload
 */
export interface AutopilotCreatedData {
  autopilot_controller_key: string;
  pod_key: string;
}

/**
 * AutopilotController terminated event payload
 */
export interface AutopilotTerminatedData {
  autopilot_controller_key: string;
  reason: string;
}

/**
 * AutopilotController thinking event payload
 * Exposes the Control Agent's decision-making process
 */
export interface AutopilotThinkingData {
  autopilot_controller_key: string;
  iteration: number;
  // Backend sends uppercase: CONTINUE, TASK_COMPLETED, NEED_HUMAN_HELP, GIVE_UP
  // Frontend prefers lowercase: continue, completed, need_help, give_up
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

/**
 * MergeRequest event payload
 */
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

/**
 * Pipeline event payload
 */
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

/**
 * Loop run event payload
 */
export interface LoopRunEventData {
  loop_id: number;
  run_id: number;
  run_number: number;
  status: string;
  pod_key?: string;
}

/**
 * Unified notification event payload (via NotificationDispatcher)
 */
export interface NotificationPayloadData {
  source: string;
  title: string;
  body: string;
  link?: string;
  priority: "normal" | "high";
  channels: Record<string, boolean>;
}

/**
 * Loop run warning event payload
 * (e.g., sandbox resume degradation)
 */
export interface LoopRunWarningData {
  loop_id: number;
  run_id: number;
  run_number: number;
  warning: string;
  detail?: string;
}

/**
 * Event handler function type
 */
export type EventHandler<T = unknown> = (event: RealtimeEvent<T>) => void;

/**
 * Connection state
 */
export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting";
