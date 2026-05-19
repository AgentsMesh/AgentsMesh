package agentpod

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

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

const (
	AgentStatusExecuting = "executing"
	AgentStatusWaiting   = "waiting"
	AgentStatusIdle      = "idle"
)

const (
	PermissionModeDefault     = "default"
	PermissionModePlan        = "plan"
	PermissionModeAcceptEdits = "acceptEdits"
	PermissionModeDontAsk     = "dontAsk"
	PermissionModeBypass      = "bypassPermissions"
)

const (
	InteractionModePTY = "pty"
	InteractionModeACP = "acp"
)

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

	Prompt string  `gorm:"column:prompt;type:text" json:"prompt,omitempty"`
	BranchName    *string `gorm:"size:255" json:"branch_name,omitempty"`
	SandboxPath   *string `gorm:"column:sandbox_path;size:500" json:"sandbox_path,omitempty"`

	Model           *string `gorm:"size:50" json:"model,omitempty"`           // opus/sonnet/haiku
	PermissionMode  *string `gorm:"size:50" json:"permission_mode,omitempty"` // default/plan/acceptEdits/dontAsk/bypassPermissions
	InteractionMode string  `gorm:"column:interaction_mode;type:varchar(10);default:pty;not null" json:"interaction_mode"`
	ErrorCode    *string `gorm:"size:100" json:"error_code,omitempty"`
	ErrorMessage *string `gorm:"type:text" json:"error_message,omitempty"`

	Title *string `gorm:"size:255" json:"title,omitempty"`

	Alias *string `gorm:"size:100" json:"alias,omitempty"`

	SessionID *string `gorm:"size:36" json:"session_id,omitempty"`

	SourcePodKey *string `gorm:"size:100" json:"source_pod_key,omitempty"`

	Perpetual     bool       `gorm:"not null;default:false" json:"perpetual"`
	RestartCount  int        `gorm:"not null;default:0" json:"restart_count"`
	LastRestartAt *time.Time `json:"last_restart_at,omitempty"`

	CredentialProfileID *int64 `json:"credential_profile_id,omitempty"`

	ResolvedConfig agent.ConfigValues `gorm:"column:config_overrides;type:jsonb;default:'{}'" json:"config_overrides,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	Runner     *runner.Runner         `gorm:"foreignKey:RunnerID" json:"runner,omitempty"`
	Agent      *agent.Agent           `gorm:"foreignKey:AgentSlug;references:Slug" json:"agent,omitempty"`
	Repository *gitprovider.Repository `gorm:"foreignKey:RepositoryID" json:"repository,omitempty"`
	Ticket     *ticket.Ticket         `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
	CreatedBy  *user.User             `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`

	Loop *PodLoopInfo `gorm:"-" json:"loop,omitempty"`
}

type PodLoopInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (Pod) TableName() string {
	return "pods"
}

func (p *Pod) IsActive() bool {
	return IsPodStatusActive(p.Status)
}

func (p *Pod) IsTerminal() bool {
	return IsPodStatusTerminal(p.Status)
}

func (p *Pod) CanReconnect() bool {
	return p.Status == StatusDisconnected
}

func (p *Pod) IsACPMode() bool {
	return p.InteractionMode == InteractionModeACP
}

func (p *Pod) GetOrganizationID() int64 {
	return p.OrganizationID
}

func (p *Pod) GetPodKey() string {
	return p.PodKey
}
