/**
 * Event types from the backend
 */
export type EventType =
  // Entity events (broadcast to organization)
  | "pod:created"
  | "pod:status_changed"
  | "pod:agent_status_changed"
  | "pod:terminated"
  | "ticket:created"
  | "ticket:updated"
  | "ticket:status_changed"
  | "ticket:moved"
  | "ticket:deleted"
  | "runner:online"
  | "runner:offline"
  | "runner:updated"
  // Notification events (targeted to specific users)
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
  identifier: string;
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
