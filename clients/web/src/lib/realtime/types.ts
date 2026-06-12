import { fromJson, type DescMessage, type MessageShape } from "@bufbuild/protobuf";

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

// Decode a realtime event's `data` field against a proto-es schema. The wire
// is `protojson` (UseProtoNames=true) so snake_case keys land here as plain
// JSON; `fromJson` rebuilds a fully-typed message instance (incl. bigint for
// int64). Unknown keys are ignored so backend can add fields without breaking
// older clients.
export function decodeEventData<Desc extends DescMessage>(
  schema: Desc,
  data: unknown,
): MessageShape<Desc> {
  return fromJson(schema, data as never, { ignoreUnknownFields: true });
}

export type {
  PodStatusChangedEventData,
  PodCreatedEventData,
  PodTitleChangedEventData,
  PodAliasChangedEventData,
  PodInitProgressEventData,
  PodRestartingEventData,
  PodPerpetualChangedEventData,
  RunnerStatusEventData,
  TicketStatusChangedEventData,
  ChannelMessageEventData,
  ChannelMessageEditedEventData,
  ChannelMessageDeletedEventData,
  ChannelMemberChangedEventData,
  AutopilotStatusChangedEventData,
  AutopilotIterationEventData,
  AutopilotCreatedEventData,
  AutopilotTerminatedEventData,
  AutopilotThinkingEventData,
  MrEventData,
  PipelineEventData,
  LoopRunEventData,
  LoopRunWarningEventData,
  NotificationPayloadEventData,
} from "@proto/events/v1/event_data_pb";

export {
  PodStatusChangedEventDataSchema,
  PodCreatedEventDataSchema,
  PodTitleChangedEventDataSchema,
  PodAliasChangedEventDataSchema,
  PodInitProgressEventDataSchema,
  PodRestartingEventDataSchema,
  PodPerpetualChangedEventDataSchema,
  RunnerStatusEventDataSchema,
  TicketStatusChangedEventDataSchema,
  ChannelMessageEventDataSchema,
  ChannelMessageEditedEventDataSchema,
  ChannelMessageDeletedEventDataSchema,
  ChannelMemberChangedEventDataSchema,
  AutopilotStatusChangedEventDataSchema,
  AutopilotIterationEventDataSchema,
  AutopilotCreatedEventDataSchema,
  AutopilotTerminatedEventDataSchema,
  AutopilotThinkingEventDataSchema,
  MrEventDataSchema,
  PipelineEventDataSchema,
  LoopRunEventDataSchema,
  LoopRunWarningEventDataSchema,
  NotificationPayloadEventDataSchema,
} from "@proto/events/v1/event_data_pb";

// Autopilot thinking — snake_case wasm-state shapes returned by
// AutopilotState.get_thinking_json (Rust serde serializes the prost struct with
// snake_case keys). Distinct from the camelCase proto event types above
// (AutopilotThinkingEventData), which carry the realtime wire payload. Mirrors
// proto/events/v1/event_data.proto AutopilotThinkingEventData field-for-field.
//
// Fields the backend omits under EmitUnpopulated:false (empty repeated fields,
// zero/empty scalars) are absent at runtime, so they are `?`-optional here —
// the type forces every consumer to guard rather than crash on `.map`/`.length`
// or render `NaN%` from `undefined * 100`.
export interface AutopilotActionData {
  type: string;
  content: string;
  reason: string;
}

export interface AutopilotProgressData {
  summary?: string;
  completed_steps?: string[];
  remaining_steps?: string[];
  percent?: number;
}

export interface AutopilotHelpSuggestionData {
  action: string;
  label: string;
}

export interface AutopilotHelpRequestData {
  reason: string;
  context: string;
  terminal_excerpt: string;
  suggestions?: AutopilotHelpSuggestionData[];
}

export interface AutopilotThinkingData {
  autopilot_controller_key: string;
  iteration: number;
  decision_type: string;
  reasoning: string;
  confidence?: number;
  action?: AutopilotActionData;
  progress?: AutopilotProgressData;
  help_request?: AutopilotHelpRequestData;
}
