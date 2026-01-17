// Package tools provides MCP tools for agent collaboration.
package tools

import (
	"context"
	"encoding/json"
)

// BindingScope represents permission scopes for pod bindings.
type BindingScope string

const (
	// ScopeTerminalRead allows reading terminal output.
	ScopeTerminalRead BindingScope = "terminal:read"
	// ScopeTerminalWrite allows writing to terminal.
	ScopeTerminalWrite BindingScope = "terminal:write"
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
	TicketStatusCanceled   TicketStatus = "canceled"
)

// TicketType represents the type of ticket.
type TicketType string

const (
	TicketTypeTask        TicketType = "task"
	TicketTypeBug         TicketType = "bug"
	TicketTypeFeature     TicketType = "feature"
	TicketTypeImprovement TicketType = "improvement"
	TicketTypeEpic        TicketType = "epic"
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

// Binding represents a pod binding.
type Binding struct {
	ID            int            `json:"id"`
	InitiatorPod  string         `json:"initiator_pod"`
	TargetPod     string         `json:"target_pod"`
	GrantedScopes []BindingScope `json:"granted_scopes"`
	PendingScopes []BindingScope `json:"pending_scopes"`
	Status        BindingStatus  `json:"status"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

// AgentTypeField can unmarshal both string and object formats of agent_type.
// Backend returns agent_type as an object {id, slug, name, ...}, but we only need the slug.
type AgentTypeField string

// UnmarshalJSON implements custom JSON unmarshaling for AgentTypeField.
func (a *AgentTypeField) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*a = AgentTypeField(str)
		return nil
	}

	// Try to unmarshal as object, extract slug
	var obj struct {
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		*a = AgentTypeField(obj.Slug)
		return nil
	}

	// Ignore unparseable data
	return nil
}

// PodCreator represents the user who created a pod.
type PodCreator struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name,omitempty"`
}

// PodTicket represents the ticket associated with a pod.
type PodTicket struct {
	ID         int    `json:"id"`
	Identifier string `json:"identifier,omitempty"`
	Title      string `json:"title"`
}

// AvailablePod represents a pod available for collaboration.
type AvailablePod struct {
	ID          int            `json:"id"`
	PodKey      string         `json:"pod_key"`
	Title       *string        `json:"title,omitempty"`
	CreatedByID int            `json:"created_by_id"`
	CreatedBy   *PodCreator    `json:"created_by,omitempty"`
	Status      PodStatus      `json:"status"`
	TicketID    *int           `json:"ticket_id,omitempty"`
	Ticket      *PodTicket     `json:"ticket,omitempty"`
	AgentType   AgentTypeField `json:"agent_type,omitempty"`
	CreatedAt   string         `json:"created_at"`
}

// GetUsername returns the username of the pod creator.
func (p *AvailablePod) GetUsername() string {
	if p.CreatedBy != nil {
		return p.CreatedBy.Username
	}
	return ""
}

// GetTicketTitle returns the title of the associated ticket.
func (p *AvailablePod) GetTicketTitle() string {
	if p.Ticket != nil {
		return p.Ticket.Title
	}
	return ""
}

// TerminalOutput represents terminal observation output.
type TerminalOutput struct {
	PodKey     string `json:"pod_key"`
	Output     string `json:"output"`
	Screen     string `json:"screen,omitempty"`
	CursorX    int    `json:"cursor_x"`
	CursorY    int    `json:"cursor_y"`
	TotalLines int    `json:"total_lines"`
	HasMore    bool   `json:"has_more"`
}

// Channel represents a collaboration channel.
type Channel struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description,omitempty"`
	ProjectID       *int    `json:"project_id,omitempty"`
	TicketID        *int    `json:"ticket_id,omitempty"`
	Document        string  `json:"document,omitempty"`
	MemberCount     int     `json:"member_count"`
	IsArchived      bool    `json:"is_archived"`
	CreatedByPod    string  `json:"created_by_pod,omitempty"`
	CreatedByUserID *int    `json:"created_by_user_id,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// ChannelMessage represents a message in a channel.
type ChannelMessage struct {
	ID           int                `json:"id"`
	ChannelID    int                `json:"channel_id"`
	SenderPod    string             `json:"sender_pod"`
	SenderUserID *int               `json:"sender_user_id,omitempty"`
	Content      string             `json:"content"`
	MessageType  ChannelMessageType `json:"message_type"`
	Mentions     []string           `json:"mentions,omitempty"`
	ReplyTo      *int               `json:"reply_to,omitempty"`
	CreatedAt    string             `json:"created_at"`
}

// Ticket represents a ticket in the system.
type Ticket struct {
	ID             int            `json:"id"`
	Identifier     string         `json:"identifier"`
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`
	Content        string         `json:"content,omitempty"`
	Type           TicketType     `json:"type"`
	Status         TicketStatus   `json:"status"`
	Priority       TicketPriority `json:"priority"`
	ProductID      int            `json:"product_id"`
	ProductName    string         `json:"product_name,omitempty"`
	ReporterID     int            `json:"reporter_id"`
	ReporterName   string         `json:"reporter_name,omitempty"`
	ParentTicketID *int           `json:"parent_ticket_id,omitempty"`
	Estimate       *int           `json:"estimate,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

// ConfigFieldSummary is a simplified config field for LLM consumption.
// Removes validation and show_when fields that are only used by frontend.
type ConfigFieldSummary struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Default  interface{} `json:"default,omitempty"`
	Options  []string    `json:"options,omitempty"`
	Required bool        `json:"required,omitempty"`
}

// AgentTypeSummary is a simplified AgentType for LLM consumption.
type AgentTypeSummary struct {
	ID          int64                  `json:"id"`
	Slug        string                 `json:"slug"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Config      []ConfigFieldSummary   `json:"config,omitempty"`
	UserConfig  map[string]interface{} `json:"user_config,omitempty"`
}

// RunnerSummary is a simplified Runner with nested Agent details.
// Optimized for LLM token efficiency - removes host_info, timestamps, etc.
type RunnerSummary struct {
	ID                int64              `json:"id"`
	NodeID            string             `json:"node_id"`
	Description       string             `json:"description,omitempty"`
	Status            string             `json:"status"`
	CurrentPods       int                `json:"current_pods"`
	MaxConcurrentPods int                `json:"max_concurrent_pods"`
	AvailableAgents   []AgentTypeSummary `json:"available_agents"`
}

// Repository represents a Git repository configuration.
type Repository struct {
	ID              int64  `json:"id"`
	ProviderType    string `json:"provider_type"`
	ProviderBaseURL string `json:"provider_base_url"`
	CloneURL        string `json:"clone_url,omitempty"`
	ExternalID      string `json:"external_id"`
	Name            string `json:"name"`
	FullPath        string `json:"full_path"`
	DefaultBranch   string `json:"default_branch"`
	TicketPrefix    string `json:"ticket_prefix,omitempty"`
	Visibility      string `json:"visibility"`
	IsActive        bool   `json:"is_active"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// PodCreateRequest represents a request to create a new pod.
type PodCreateRequest struct {
	RunnerID      int    `json:"runner_id,omitempty"`
	AgentTypeID   *int64 `json:"agent_type_id,omitempty"` // Required by backend API
	TicketID      *int   `json:"ticket_id,omitempty"`
	InitialPrompt string `json:"initial_prompt,omitempty"`
	Model         string `json:"model,omitempty"`
}

// PodCreateResponse represents the response from creating a pod.
type PodCreateResponse struct {
	PodKey      string `json:"pod_key"`
	Status      string `json:"status"`
	TerminalURL string `json:"terminal_url,omitempty"`
}

// TerminalClient defines the interface for terminal operations.
type TerminalClient interface {
	ObserveTerminal(ctx context.Context, podKey string, lines int, raw bool, includeScreen bool) (*TerminalOutput, error)
	SendTerminalText(ctx context.Context, podKey string, text string) error
	SendTerminalKey(ctx context.Context, podKey string, keys []string) error
}

// DiscoveryClient defines the interface for pod discovery.
type DiscoveryClient interface {
	ListAvailablePods(ctx context.Context) ([]AvailablePod, error)
	ListRunners(ctx context.Context) ([]RunnerSummary, error)
	ListRepositories(ctx context.Context) ([]Repository, error)
}

// BindingClient defines the interface for pod binding operations.
type BindingClient interface {
	RequestBinding(ctx context.Context, targetPod string, scopes []BindingScope) (*Binding, error)
	AcceptBinding(ctx context.Context, bindingID int) (*Binding, error)
	RejectBinding(ctx context.Context, bindingID int, reason string) (*Binding, error)
	UnbindPod(ctx context.Context, targetPod string) error
	GetBindings(ctx context.Context, status *BindingStatus) ([]Binding, error)
	GetBoundPods(ctx context.Context) ([]string, error)
}

// ChannelClient defines the interface for channel operations.
type ChannelClient interface {
	SearchChannels(ctx context.Context, name string, projectID, ticketID *int, isArchived *bool, offset, limit int) ([]Channel, error)
	CreateChannel(ctx context.Context, name, description string, projectID, ticketID *int) (*Channel, error)
	GetChannel(ctx context.Context, channelID int) (*Channel, error)
	SendMessage(ctx context.Context, channelID int, content string, msgType ChannelMessageType, mentions []string, replyTo *int) (*ChannelMessage, error)
	GetMessages(ctx context.Context, channelID int, beforeTime, afterTime *string, mentionedPod *string, limit int) ([]ChannelMessage, error)
	GetDocument(ctx context.Context, channelID int) (string, error)
	UpdateDocument(ctx context.Context, channelID int, document string) error
}

// TicketClient defines the interface for ticket operations.
type TicketClient interface {
	SearchTickets(ctx context.Context, productID *int, status *TicketStatus, ticketType *TicketType, priority *TicketPriority, assigneeID, parentID *int, query string, limit, page int) ([]Ticket, error)
	GetTicket(ctx context.Context, ticketID string) (*Ticket, error)
	CreateTicket(ctx context.Context, productID int, title, description string, ticketType TicketType, priority TicketPriority, parentTicketID *int) (*Ticket, error)
	UpdateTicket(ctx context.Context, ticketID string, title, description *string, status *TicketStatus, priority *TicketPriority, ticketType *TicketType) (*Ticket, error)
}

// PodClient defines the interface for pod creation.
type PodClient interface {
	CreatePod(ctx context.Context, req *PodCreateRequest) (*PodCreateResponse, error)
}

// CollaborationClient combines all collaboration interfaces.
type CollaborationClient interface {
	TerminalClient
	DiscoveryClient
	BindingClient
	ChannelClient
	TicketClient
	PodClient

	// GetPodKey returns the current pod's key.
	GetPodKey() string
}
