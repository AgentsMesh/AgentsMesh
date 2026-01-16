package agentpod

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

// Pod status constants
const (
	StatusInitializing = "initializing"
	StatusRunning      = "running"
	StatusPaused       = "paused"
	StatusDisconnected = "disconnected" // User closed browser
	StatusOrphaned     = "orphaned"     // Lost due to runner restart
	StatusCompleted    = "completed"
	StatusTerminated   = "terminated"
	StatusError        = "error"

	// Legacy aliases (for backward compatibility during transition)
	PodStatusInitializing = StatusInitializing
	PodStatusRunning      = StatusRunning
	PodStatusPaused       = StatusPaused
	PodStatusDisconnected = StatusDisconnected
	PodStatusOrphaned     = StatusOrphaned
	PodStatusTerminated   = StatusTerminated
	PodStatusError        = StatusError
)

// Agent status constants
const (
	AgentStatusUnknown  = "unknown"
	AgentStatusIdle     = "idle"
	AgentStatusWorking  = "working"
	AgentStatusWaiting  = "waiting"
	AgentStatusFinished = "finished"
)

// Think level magic words for Claude
const (
	ThinkLevelNone       = ""
	ThinkLevelUltrathink = "ultrathink"
	ThinkLevelMegathink  = "megathink"
)

// Permission mode for Claude
const (
	PermissionModePlan    = "plan"
	PermissionModeDefault = "default"
	PermissionModeBypass  = "bypassPermissions"
)

// Pod represents an AI coding pod (AgentPod instance)
type Pod struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	PodKey   string `gorm:"size:100;not null;uniqueIndex" json:"pod_key"`
	RunnerID int64  `gorm:"not null;index" json:"runner_id"`

	AgentTypeID       *int64 `json:"agent_type_id,omitempty"`
	CustomAgentTypeID *int64 `json:"custom_agent_type_id,omitempty"`

	RepositoryID *int64 `json:"repository_id,omitempty"`
	TicketID     *int64 `json:"ticket_id,omitempty"`
	CreatedByID  int64  `gorm:"not null" json:"created_by_id"`

	PtyPID      *int   `gorm:"column:pty_pid" json:"pty_pid,omitempty"`
	Status      string `gorm:"size:50;not null;default:'initializing';index" json:"status"`
	AgentStatus string `gorm:"size:50;not null;default:'unknown'" json:"agent_status"`
	AgentPID    *int   `gorm:"column:agent_pid" json:"agent_pid,omitempty"` // Claude/Agent process PID

	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	LastActivity *time.Time `json:"last_activity,omitempty"`

	// Initial prompt and configuration
	InitialPrompt string  `gorm:"type:text" json:"initial_prompt,omitempty"`
	BranchName    *string `gorm:"size:255" json:"branch_name,omitempty"`
	WorktreePath  *string `gorm:"size:500" json:"worktree_path,omitempty"`

	// Agent configuration used for this pod
	Model          *string `gorm:"size:50" json:"model,omitempty"`           // opus/sonnet/haiku
	PermissionMode *string `gorm:"size:50" json:"permission_mode,omitempty"` // plan/default/bypassPermissions
	ThinkLevel     *string `gorm:"size:50" json:"think_level,omitempty"`     // ultrathink/megathink

	// Terminal title from OSC 0/2 escape sequences
	Title *string `gorm:"size:255" json:"title,omitempty"`

	// ConfigOverrides stores pod-level configuration overrides
	// Merged with organization defaults during Pod creation
	ConfigOverrides agent.ConfigValues `gorm:"type:jsonb;default:'{}'" json:"config_overrides,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	Runner          *runner.Runner         `gorm:"foreignKey:RunnerID" json:"runner,omitempty"`
	AgentType       *agent.AgentType       `gorm:"foreignKey:AgentTypeID" json:"agent_type,omitempty"`
	CustomAgentType *agent.CustomAgentType `gorm:"foreignKey:CustomAgentTypeID" json:"custom_agent_type,omitempty"`
	Ticket          *ticket.Ticket         `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
	CreatedBy       *user.User             `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
}

func (Pod) TableName() string {
	return "pods"
}

// IsActive returns true if pod is active
func (p *Pod) IsActive() bool {
	return p.Status == PodStatusRunning ||
		p.Status == PodStatusInitializing ||
		p.Status == PodStatusPaused ||
		p.Status == PodStatusDisconnected
}

// IsTerminal returns true if pod is in a terminal state
func (p *Pod) IsTerminal() bool {
	return p.Status == PodStatusTerminated ||
		p.Status == PodStatusOrphaned ||
		p.Status == PodStatusError
}

// CanReconnect returns true if pod can be reconnected
func (p *Pod) CanReconnect() bool {
	return p.Status == PodStatusDisconnected
}

// GetOrganizationID returns the organization ID (implements middleware.PodGetter)
func (p *Pod) GetOrganizationID() int64 {
	return p.OrganizationID
}

// GetPodKey returns the pod key (implements middleware.PodGetter)
func (p *Pod) GetPodKey() string {
	return p.PodKey
}

// PreparationConfig holds the preparation script configuration
type PreparationConfig struct {
	Script  string `json:"script,omitempty"`
	Timeout int    `json:"timeout,omitempty"` // in seconds
}

// CreatePodCommand represents a command to create a pod on a runner
type CreatePodCommand struct {
	PodKey            string             `json:"pod_id"` // Use pod_id for compatibility with runner
	InitialCommand    string             `json:"initial_command,omitempty"`
	InitialPrompt     string             `json:"initial_prompt,omitempty"`
	PermissionMode    string             `json:"permission_mode,omitempty"`
	TicketIdentifier  string             `json:"ticket_identifier,omitempty"`
	WorktreeSuffix    string             `json:"worktree_suffix,omitempty"`
	EnvVars           map[string]string  `json:"env_vars,omitempty"`
	PreparationConfig *PreparationConfig `json:"preparation_config,omitempty"`
}

// TerminatePodCommand represents a command to terminate a pod
type TerminatePodCommand struct {
	PodKey string `json:"pod_id"`
}
