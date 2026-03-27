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
	// CategoryNotification represents notification events (targeted to specific users)
	CategoryNotification EventCategory = "notification"
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
)

// ===== Notification Events (Category: notification) =====
const (
	EventPodNotification EventType = "pod:notification" // OSC 777
	EventTaskCompleted        EventType = "task:completed"        // Agent finished
	EventMentionNotification  EventType = "mention:notification"  // @mention (future)
	EventNotification         EventType = "notification"          // Unified notification (via dispatcher)
)

// NotificationPayload is the unified payload for all dispatched notifications
type NotificationPayload struct {
	Source   string          `json:"source"`
	Title    string          `json:"title"`
	Body     string          `json:"body"`
	Link     string          `json:"link,omitempty"`
	Priority string          `json:"priority"`
	Channels map[string]bool `json:"channels"`
}

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

	// TargetUserID is the target user for notification events (single user)
	TargetUserID *int64 `json:"target_user_id,omitempty"`
	// TargetUserIDs is the target users for notification events (multiple users)
	TargetUserIDs []int64 `json:"target_user_ids,omitempty"`

	// EntityType is the type of entity (pod, ticket, runner, channel)
	EntityType string `json:"entity_type,omitempty"`
	// EntityID is the unique identifier of the entity
	EntityID string `json:"entity_id,omitempty"`

	// Data contains the event-specific payload
	Data json.RawMessage `json:"data"`
	// Timestamp is the Unix millisecond timestamp when the event was created
	Timestamp int64 `json:"timestamp"`

	// SourceInstanceID identifies the server instance that published this event
	// Used to prevent duplicate dispatch when receiving from Redis
	SourceInstanceID string `json:"source_instance_id,omitempty"`
}

// EventHandler is a function that handles events
type EventHandler func(event *Event)
