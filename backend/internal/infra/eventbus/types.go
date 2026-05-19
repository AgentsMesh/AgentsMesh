package eventbus

import (
	"encoding/json"
)

type EventType string

type EventCategory string

const (
	CategoryEntity EventCategory = "entity"
	CategorySystem EventCategory = "system"
)

const (
	EventPodCreated       EventType = "pod:created"
	EventPodStatusChanged EventType = "pod:status_changed"
	EventPodAgentChanged  EventType = "pod:agent_status_changed"
	EventPodTerminated    EventType = "pod:terminated"
	EventPodTitleChanged  EventType = "pod:title_changed"
	EventPodAliasChanged  EventType = "pod:alias_changed"
	EventPodInitProgress  EventType = "pod:init_progress"
	EventPodRestarting        EventType = "pod:restarting"
	EventPodPerpetualChanged  EventType = "pod:perpetual_changed"

	EventChannelMessage        EventType = "channel:message"
	EventChannelMessageEdited  EventType = "channel:message_edited"
	EventChannelMessageDeleted EventType = "channel:message_deleted"

	EventTicketCreated       EventType = "ticket:created"
	EventTicketUpdated       EventType = "ticket:updated"
	EventTicketStatusChanged EventType = "ticket:status_changed"
	EventTicketMoved         EventType = "ticket:moved"
	EventTicketDeleted       EventType = "ticket:deleted"

	EventRunnerOnline  EventType = "runner:online"
	EventRunnerOffline EventType = "runner:offline"
	EventRunnerUpdated EventType = "runner:updated"

	EventAutopilotStatusChanged EventType = "autopilot:status_changed"
	EventAutopilotIteration     EventType = "autopilot:iteration"
	EventAutopilotCreated       EventType = "autopilot:created"
	EventAutopilotTerminated    EventType = "autopilot:terminated"
	EventAutopilotThinking      EventType = "autopilot:thinking"

	EventMRCreated EventType = "mr:created"
	EventMRUpdated EventType = "mr:updated"
	EventMRMerged  EventType = "mr:merged"
	EventMRClosed  EventType = "mr:closed"

	EventPipelineUpdated EventType = "pipeline:updated"

	EventLoopRunStarted   EventType = "loop_run:started"
	EventLoopRunCompleted EventType = "loop_run:completed"
	EventLoopRunFailed    EventType = "loop_run:failed"
	EventLoopRunWarning   EventType = "loop_run:warning"
	EventChannelMemberAdded   EventType = "channel:member_added"
	EventChannelMemberRemoved EventType = "channel:member_removed"

	EventBlockstoreOp EventType = "blockstore:op"
)

const (
	EventSystemMaintenance EventType = "system:maintenance"
)

type Event struct {
	Type EventType `json:"type"`
	Category EventCategory `json:"category"`
	OrganizationID int64 `json:"organization_id"`

	EntityType string `json:"entity_type,omitempty"`
	EntityID string `json:"entity_id,omitempty"`

	Data json.RawMessage `json:"data"`
	Timestamp int64 `json:"timestamp"`

	// TargetUserIDs restricts delivery to specific users instead of broadcasting to the org.
	// When non-empty, HubEventSubscriber uses SendToUser per user instead of BroadcastToOrg.
	TargetUserIDs []int64 `json:"target_user_ids,omitempty"`

	SourceInstanceID string `json:"source_instance_id,omitempty"`
}

type EventHandler func(event *Event)
