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

	// Channel events
	EventChannelMessage EventType = "channel:message"

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
)

// ===== Notification Events (Category: notification) =====
const (
	EventTerminalNotification EventType = "terminal:notification" // OSC 777
	EventTaskCompleted        EventType = "task:completed"        // Agent finished
	EventMentionNotification  EventType = "mention:notification"  // @mention (future)
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

// PodStatusChangedData represents the payload for pod status change events
type PodStatusChangedData struct {
	PodKey         string `json:"pod_key"`
	Status         string `json:"status"`
	PreviousStatus string `json:"previous_status,omitempty"`
	AgentStatus    string `json:"agent_status,omitempty"`
}

// PodCreatedData represents the payload for pod created events
type PodCreatedData struct {
	PodKey      string `json:"pod_key"`
	Status      string `json:"status"`
	AgentStatus string `json:"agent_status,omitempty"`
	RunnerID    int64  `json:"runner_id"`
	TicketID    *int64 `json:"ticket_id,omitempty"`
	CreatedByID int64  `json:"created_by_id"`
}

// RunnerStatusData represents the payload for runner status events
type RunnerStatusData struct {
	RunnerID      int64  `json:"runner_id"`
	NodeID        string `json:"node_id"`
	Status        string `json:"status"`
	CurrentPods   int    `json:"current_pods,omitempty"`
	LastHeartbeat string `json:"last_heartbeat,omitempty"`
}

// TicketStatusChangedData represents the payload for ticket status change events
type TicketStatusChangedData struct {
	Identifier     string `json:"identifier"`
	Status         string `json:"status"`
	PreviousStatus string `json:"previous_status,omitempty"`
}

// TerminalNotificationData represents the payload for terminal notification events
type TerminalNotificationData struct {
	PodKey string `json:"pod_key"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// TaskCompletedData represents the payload for task completed events
type TaskCompletedData struct {
	PodKey      string `json:"pod_key"`
	AgentStatus string `json:"agent_status"`
	TicketID    *int64 `json:"ticket_id,omitempty"`
}

// PodTitleChangedData represents the payload for pod title change events
type PodTitleChangedData struct {
	PodKey string `json:"pod_key"`
	Title  string `json:"title"`
}

// ChannelMessageData represents the payload for channel message events
type ChannelMessageData struct {
	ID           int64             `json:"id"`
	ChannelID    int64             `json:"channel_id"`
	SenderPod    *string           `json:"sender_pod,omitempty"`
	SenderUserID *int64            `json:"sender_user_id,omitempty"`
	MessageType  string            `json:"message_type"`
	Content      string            `json:"content"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
	CreatedAt    string            `json:"created_at"`
}
