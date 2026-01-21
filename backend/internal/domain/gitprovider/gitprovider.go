package gitprovider

import (
	"time"
)

// Provider types (used by both user-level repository providers and repositories)
const (
	ProviderTypeGitHub = "github"
	ProviderTypeGitLab = "gitlab"
	ProviderTypeGitee  = "gitee"
	ProviderTypeSSH    = "ssh" // SSH-based Git server (no API)
)

// NOTE: Organization-level GitProvider has been removed.
// Git providers are now managed at the user level via:
// - UserRepositoryProvider (for importing repositories)
// - UserGitCredential (for Git operations)
// See: /backend/internal/domain/user/repository_provider.go
//      /backend/internal/domain/user/git_credential.go

// Repository represents a Git repository configured in the system
// Self-contained design: repository stores all necessary info, no git_provider_id dependency
type Repository struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	// Provider info (self-contained, no foreign key to git_providers)
	ProviderType    string `gorm:"size:50;not null" json:"provider_type"`      // github, gitlab, gitee, generic
	ProviderBaseURL string `gorm:"size:255;not null" json:"provider_base_url"` // https://github.com, https://gitlab.company.com
	CloneURL        string `gorm:"size:500" json:"clone_url"`                  // Full clone URL

	ExternalID    string  `gorm:"size:255;not null" json:"external_id"`
	Name          string  `gorm:"size:255;not null" json:"name"`
	FullPath      string  `gorm:"size:500;not null" json:"full_path"`
	DefaultBranch string  `gorm:"size:100;default:'main'" json:"default_branch"`
	TicketPrefix  *string `gorm:"size:10" json:"ticket_prefix,omitempty"`

	// Visibility: "organization" (all members can see), "private" (only importer can see)
	Visibility       string `gorm:"size:20;not null;default:'organization'" json:"visibility"`
	ImportedByUserID *int64 `gorm:"index" json:"imported_by_user_id,omitempty"` // User who imported this repo

	// Workspace preparation
	PreparationScript  *string `gorm:"type:text" json:"preparation_script,omitempty"`    // Script to run after worktree creation
	PreparationTimeout *int    `gorm:"default:300" json:"preparation_timeout,omitempty"` // Script timeout in seconds (default 300)

	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"` // Soft delete support
}

func (Repository) TableName() string {
	return "repositories"
}
