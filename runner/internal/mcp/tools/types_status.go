// Package tools provides MCP tools for agent collaboration.
package tools

import (
	"encoding/json"
)

// BindingScope represents permission scopes for pod bindings.
type BindingScope string

const (
	// ScopePodRead allows reading pod output.
	ScopePodRead BindingScope = "pod:read"
	// ScopePodWrite allows writing to pod.
	ScopePodWrite BindingScope = "pod:write"
)

// BindingStatus represents the status of a pod binding.
type BindingStatus string

const (
	BindingStatusPending  BindingStatus = "pending"
	BindingStatusActive   BindingStatus = "active"
	BindingStatusRejected BindingStatus = "rejected"
	BindingStatusInactive BindingStatus = "inactive"
	BindingStatusExpired  BindingStatus = "expired"
)

// PodStatus represents the status of a pod.
type PodStatus string

const (
	PodStatusInitializing PodStatus = "initializing"
	PodStatusRunning      PodStatus = "running"
	PodStatusDisconnected PodStatus = "disconnected"
	PodStatusCompleted    PodStatus = "completed"
	PodStatusError        PodStatus = "error"
	PodStatusOrphaned     PodStatus = "orphaned"
)

// TicketStatus represents ticket workflow states.
type TicketStatus string

const (
	TicketStatusBacklog    TicketStatus = "backlog"
	TicketStatusTodo       TicketStatus = "todo"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusInReview   TicketStatus = "in_review"
	TicketStatusDone       TicketStatus = "done"
)

// TicketPriority represents ticket priority levels.
type TicketPriority string

const (
	TicketPriorityUrgent TicketPriority = "urgent"
	TicketPriorityHigh   TicketPriority = "high"
	TicketPriorityMedium TicketPriority = "medium"
	TicketPriorityLow    TicketPriority = "low"
	TicketPriorityNone   TicketPriority = "none"
)

// ChannelMessageType represents the type of channel message.
type ChannelMessageType string

const (
	ChannelMessageTypeText   ChannelMessageType = "text"
	ChannelMessageTypeSystem ChannelMessageType = "system"
)

// AgentField can unmarshal both string and object formats of agent.
// Backend returns agent as an object {id, slug, name, ...}, but we only need the slug.
type AgentField string

// UnmarshalJSON implements custom JSON unmarshaling for AgentField.
func (a *AgentField) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*a = AgentField(str)
		return nil
	}

	// Try to unmarshal as object, extract slug
	var obj struct {
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		*a = AgentField(obj.Slug)
		return nil
	}

	// Ignore unparseable data
	return nil
}
