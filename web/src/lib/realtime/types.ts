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
  | "pod:alias_changed"
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
  | "pod:notification"
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

// Re-export all event payload types from split files
export type {
  PodStatusChangedData,
  PodCreatedData,
  RunnerStatusData,
  TicketStatusChangedData,
  TerminalNotificationData,
  TaskCompletedData,
  PodTitleChangedData,
  PodAliasChangedData,
  PodInitProgressData,
  ChannelMessageData,
  ChannelMessageEditedData,
  ChannelMessageDeletedData,
} from "./entityEventTypes";

export type {
  AutopilotStatusChangedData,
  AutopilotIterationData,
  AutopilotCreatedData,
  AutopilotTerminatedData,
  AutopilotThinkingData,
  MREventData,
  PipelineEventData,
  LoopRunEventData,
  NotificationPayloadData,
  LoopRunWarningData,
} from "./featureEventTypes";
