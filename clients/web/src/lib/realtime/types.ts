export type EventType =
  | "pod:created"
  | "pod:status_changed"
  | "pod:agent_status_changed"
  | "pod:terminated"
  | "pod:title_changed"
  | "pod:alias_changed"
  | "pod:init_progress"
  | "pod:restarting"
  | "pod:perpetual_changed"
  | "channel:message"
  | "channel:message_edited"
  | "channel:message_deleted"
  | "channel:member_added"
  | "channel:member_removed"
  | "ticket:created"
  | "ticket:updated"
  | "ticket:status_changed"
  | "ticket:moved"
  | "ticket:deleted"
  | "runner:online"
  | "runner:offline"
  | "runner:updated"
  | "autopilot:status_changed"
  | "autopilot:iteration"
  | "autopilot:created"
  | "autopilot:terminated"
  | "autopilot:thinking"
  | "mr:created"
  | "mr:updated"
  | "mr:merged"
  | "mr:closed"
  | "pipeline:updated"
  | "loop_run:started"
  | "loop_run:completed"
  | "loop_run:failed"
  | "loop_run:warning"
  | "blockstore:op"
  | "notification"
  | "system:maintenance"
  | "connected"
  | "ping"
  | "pong";

export type EventCategory = "entity" | "notification" | "system";

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

export type EventHandler<T = unknown> = (event: RealtimeEvent<T>) => void;

export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting";

export type {
  PodStatusChangedData,
  PodCreatedData,
  RunnerStatusData,
  TicketStatusChangedData,
  PodTitleChangedData,
  PodAliasChangedData,
  PodPerpetualChangedData,
  PodInitProgressData,
  ChannelMessageData,
  ChannelMessageEditedData,
  ChannelMessageDeletedData,
  ChannelMemberChangedData,
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
