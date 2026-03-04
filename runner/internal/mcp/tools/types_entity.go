// Package tools provides MCP tools for agent collaboration.
package tools

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

// PodCreator represents the user who created a pod.
type PodCreator struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name,omitempty"`
}

// PodTicket represents the ticket associated with a pod.
type PodTicket struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug,omitempty"`
	Title string `json:"title"`
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
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	TicketSlug   string `json:"ticket_slug,omitempty"`
	Document     string `json:"document,omitempty"`
	MemberCount  int    `json:"member_count"`
	IsArchived   bool   `json:"is_archived"`
	CreatedByPod string `json:"created_by_pod,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
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
	Slug              string         `json:"slug"`
	Title             string         `json:"title"`
	Content           string         `json:"content,omitempty"`
	Status            TicketStatus   `json:"status"`
	Priority          TicketPriority `json:"priority"`
	ProductName       string         `json:"product_name,omitempty"`
	ReporterName      string         `json:"reporter_name,omitempty"`
	ParentTicketSlug  string         `json:"parent_ticket_slug,omitempty"`
	Estimate          *int           `json:"estimate,omitempty"`
	ContentTotalLines int            `json:"content_total_lines,omitempty"` // Total lines of converted content
	ContentOffset     int            `json:"content_offset,omitempty"`      // Start line of this response (0-based)
	ContentLimit      int            `json:"content_limit,omitempty"`       // Number of lines returned
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

// TicketComment represents a comment on a ticket.
type TicketComment struct {
	ID        int64  `json:"id"`
	TicketID  int64  `json:"ticket_id"`
	Content   string `json:"content"`
	ParentID  *int64 `json:"parent_id,omitempty"`
	AuthorID  int64  `json:"author_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
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
	RunnerID            int                    `json:"runner_id,omitempty"`
	AgentTypeID         *int64                 `json:"agent_type_id,omitempty"` // Required by backend API
	TicketSlug          *string                `json:"ticket_slug,omitempty"`
	InitialPrompt       string                 `json:"initial_prompt,omitempty"`
	Model               string                 `json:"model,omitempty"`
	RepositoryID        *int64                 `json:"repository_id,omitempty"`        // Repository ID (mutually exclusive with repository_url)
	RepositoryURL       *string                `json:"repository_url,omitempty"`       // Direct repository URL (takes precedence over repository_id)
	BranchName          *string                `json:"branch_name,omitempty"`          // Git branch name
	CredentialProfileID *int64                 `json:"credential_profile_id,omitempty"` // Credential profile ID (0 or nil = RunnerHost mode)
	ConfigOverrides     map[string]interface{} `json:"config_overrides,omitempty"`     // Override agent type default configuration
	PermissionMode      *string                `json:"permission_mode,omitempty"`      // "plan", "default", or "bypassPermissions"
}

// PodCreateResponse represents the response from creating a pod.
type PodCreateResponse struct {
	PodKey      string `json:"pod_key"`
	Status      string `json:"status"`
	TerminalURL string `json:"terminal_url,omitempty"`
}
