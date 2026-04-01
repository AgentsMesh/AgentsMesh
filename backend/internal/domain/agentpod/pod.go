package agentpod

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
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
)

// Agent status constants
const (
	AgentStatusExecuting = "executing"
	AgentStatusWaiting   = "waiting"
	AgentStatusIdle      = "idle"
)

// Permission mode for Claude
const (
	PermissionModePlan    = "plan"
	PermissionModeDefault = "default"
	PermissionModeBypass  = "bypassPermissions"
)

// Interaction mode constants
const (
	InteractionModePTY = "pty"
	InteractionModeACP = "acp"
)

// Pod represents an AI coding pod (AgentPod instance)
type Pod struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	PodKey   string `gorm:"size:100;not null;uniqueIndex" json:"pod_key"`
	RunnerID int64  `gorm:"not null;index" json:"runner_id"`

	AgentSlug string `gorm:"size:100;column:agent_slug" json:"agent_slug,omitempty"`

	RepositoryID *int64 `json:"repository_id,omitempty"`
	TicketID     *int64 `json:"ticket_id,omitempty"`
	CreatedByID  int64  `gorm:"not null" json:"created_by_id"`

	TerminalPID *int   `gorm:"column:pty_pid" json:"pty_pid,omitempty"`
	Status      string `gorm:"size:50;not null;default:'initializing';index" json:"status"`
	AgentStatus string `gorm:"size:50;not null;default:'idle'" json:"agent_status"`
	AgentPID    *int   `gorm:"column:agent_pid" json:"agent_pid,omitempty"` // Claude/Agent process PID

	StartedAt         *time.Time `json:"started_at,omitempty"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
	LastActivity      *time.Time `json:"last_activity,omitempty"`
	AgentWaitingSince *time.Time `json:"-"`

	// Initial prompt and configuration
	InitialPrompt string  `gorm:"type:text" json:"initial_prompt,omitempty"`
	BranchName    *string `gorm:"size:255" json:"branch_name,omitempty"`
	SandboxPath   *string `gorm:"column:sandbox_path;size:500" json:"sandbox_path,omitempty"`

	// Agent configuration used for this pod
	Model           *string `gorm:"size:50" json:"model,omitempty"`           // opus/sonnet/haiku
	PermissionMode  *string `gorm:"size:50" json:"permission_mode,omitempty"` // plan/default/bypassPermissions
	InteractionMode string  `gorm:"column:interaction_mode;type:varchar(10);default:pty;not null" json:"interaction_mode"`
	// Error details from Runner (e.g., git clone auth failure)
	ErrorCode    *string `gorm:"size:100" json:"error_code,omitempty"`
	ErrorMessage *string `gorm:"type:text" json:"error_message,omitempty"`

	// Terminal title from OSC 0/2 escape sequences
	Title *string `gorm:"size:255" json:"title,omitempty"`

	// User-assigned alias for display purposes
	Alias *string `gorm:"size:100" json:"alias,omitempty"`

	// Session ID for agent session management (e.g., Claude Code --session-id)
	// Used for resume functionality - allows agents to restore conversation context
	SessionID *string `gorm:"size:36" json:"session_id,omitempty"`

	// SourcePodKey tracks the original pod when this pod was created via resume
	// Enables tracking the chain of resumed sessions
	SourcePodKey *string `gorm:"size:100" json:"source_pod_key,omitempty"`

	// CredentialProfileID records which credential profile was used to create this pod.
	// nil = used default profile (or RunnerHost fallback), >0 = specific profile ID.
	// Sentinel value 0 is NOT stored (FK constraint); explicit RunnerHost is stored as nil.
	CredentialProfileID *int64 `json:"credential_profile_id,omitempty"`

	// ConfigOverrides stores pod-level configuration overrides
	// Merged with organization defaults during Pod creation
	ConfigOverrides agent.ConfigValues `gorm:"type:jsonb;default:'{}'" json:"config_overrides,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	Runner     *runner.Runner         `gorm:"foreignKey:RunnerID" json:"runner,omitempty"`
	Agent      *agent.Agent           `gorm:"foreignKey:AgentSlug;references:Slug" json:"agent,omitempty"`
	Repository *gitprovider.Repository `gorm:"foreignKey:RepositoryID" json:"repository,omitempty"`
	Ticket     *ticket.Ticket         `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
	CreatedBy  *user.User             `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`

	// Virtual field: populated by service layer via loop_runs join, not a DB column
	Loop *PodLoopInfo `gorm:"-" json:"loop,omitempty"`
}

// PodLoopInfo contains minimal loop information for pod display.
type PodLoopInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (Pod) TableName() string {
	return "pods"
}

// IsActive returns true if pod is active
func (p *Pod) IsActive() bool {
	return IsPodStatusActive(p.Status)
}

// IsTerminal returns true if pod is in a terminal state
func (p *Pod) IsTerminal() bool {
	return IsPodStatusTerminal(p.Status)
}

// CanReconnect returns true if pod can be reconnected
func (p *Pod) CanReconnect() bool {
	return p.Status == StatusDisconnected
}

// IsACPMode returns true if the pod uses ACP interaction mode.
func (p *Pod) IsACPMode() bool {
	return p.InteractionMode == InteractionModeACP
}

// GetOrganizationID returns the organization ID (implements middleware.PodGetter)
func (p *Pod) GetOrganizationID() int64 {
	return p.OrganizationID
}

// GetPodKey returns the pod key (implements middleware.PodGetter)
func (p *Pod) GetPodKey() string {
	return p.PodKey
}
