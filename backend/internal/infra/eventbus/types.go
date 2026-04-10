package eventbus

import (
	"encoding/json"
)

// EventType defines the type of event
type EventType string

// EventCategory defines the category of event for routing
type EventCategory string

const (
	// CategoryEntity represents entity state change events (broadcast to org)
	CategoryEntity EventCategory = "entity"
	// CategorySystem represents system-level events
	CategorySystem EventCategory = "system"
)

// ===== Entity Events (Category: entity) =====
const (
	// Pod events
	EventPodCreated       EventType = "pod:created"
	EventPodStatusChanged EventType = "pod:status_changed"
	EventPodAgentChanged  EventType = "pod:agent_status_changed"
	EventPodTerminated    EventType = "pod:terminated"
	EventPodTitleChanged  EventType = "pod:title_changed"
	EventPodAliasChanged  EventType = "pod:alias_changed"
	EventPodInitProgress  EventType = "pod:init_progress"
	EventPodRestarting        EventType = "pod:restarting"
	EventPodPerpetualChanged  EventType = "pod:perpetual_changed"

	// Channel events
	EventChannelMessage        EventType = "channel:message"
	EventChannelMessageEdited  EventType = "channel:message_edited"
	EventChannelMessageDeleted EventType = "channel:message_deleted"

	// Ticket events
	EventTicketCreated       EventType = "ticket:created"
	EventTicketUpdated       EventType = "ticket:updated"
	EventTicketStatusChanged EventType = "ticket:status_changed"
	EventTicketMoved         EventType = "ticket:moved"
	EventTicketDeleted       EventType = "ticket:deleted"

	// Runner events
	EventRunnerOnline  EventType = "runner:online"
	EventRunnerOffline EventType = "runner:offline"
	EventRunnerUpdated EventType = "runner:updated"

	// AutopilotController events
	EventAutopilotStatusChanged EventType = "autopilot:status_changed"
	EventAutopilotIteration     EventType = "autopilot:iteration"
	EventAutopilotCreated       EventType = "autopilot:created"
	EventAutopilotTerminated    EventType = "autopilot:terminated"
	EventAutopilotThinking      EventType = "autopilot:thinking"

	// MergeRequest events
	EventMRCreated EventType = "mr:created"
	EventMRUpdated EventType = "mr:updated"
	EventMRMerged  EventType = "mr:merged"
	EventMRClosed  EventType = "mr:closed"

	// Pipeline events
	EventPipelineUpdated EventType = "pipeline:updated"

	// Loop events
	EventLoopRunStarted   EventType = "loop_run:started"
	EventLoopRunCompleted EventType = "loop_run:completed"
	EventLoopRunFailed    EventType = "loop_run:failed"
	EventLoopRunWarning   EventType = "loop_run:warning"
	// Channel membership events
	EventChannelMemberAdded   EventType = "channel:member_added"
	EventChannelMemberRemoved EventType = "channel:member_removed"
)

// ===== System Events (Category: system) =====
const (
	EventSystemMaintenance EventType = "system:maintenance"
)

// Event represents a unified event structure
type Event struct {
	// Type is the event type identifier
	Type EventType `json:"type"`
	// Category determines the routing strategy (broadcast vs targeted)
	Category EventCategory `json:"category"`
	// OrganizationID is the organization this event belongs to
	OrganizationID int64 `json:"organization_id"`

	// EntityType is the type of entity (pod, ticket, runner, channel)
	EntityType string `json:"entity_type,omitempty"`
	// EntityID is the unique identifier of the entity
	EntityID string `json:"entity_id,omitempty"`

	// Data contains the event-specific payload
	Data json.RawMessage `json:"data"`
	// Timestamp is the Unix millisecond timestamp when the event was created
	Timestamp int64 `json:"timestamp"`

	// TargetUserIDs restricts delivery to specific users instead of broadcasting to the org.
	// When non-empty, HubEventSubscriber uses SendToUser per user instead of BroadcastToOrg.
	TargetUserIDs []int64 `json:"target_user_ids,omitempty"`

	// SourceInstanceID identifies the server instance that published this event
	SourceInstanceID string `json:"source_instance_id,omitempty"`
}

// EventHandler is a function that handles events
type EventHandler func(event *Event)
