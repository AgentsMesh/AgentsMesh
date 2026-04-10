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
 * Pod title changed event payload (OSC 0/2)
 */
export interface PodTitleChangedData {
  pod_key: string;
  title: string;
}

/**
 * Pod alias changed event payload
 */
export interface PodAliasChangedData {
  pod_key: string;
  alias: string | null;
}

/**
 * Pod initialization progress event payload
 */
export interface PodInitProgressData {
  pod_key: string;
  phase: string; // pending, cloning, preparing, starting_pod, ready
  progress: number; // 0-100
  message: string; // Human-readable progress message
}

/**
 * Pod perpetual mode changed event payload
 */
export interface PodPerpetualChangedData {
  pod_key: string;
  perpetual: boolean;
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
 * Channel member changed event payload
 */
export interface ChannelMemberChangedData {
  channel_id: number;
  user_id: number;
  role?: string;
}
