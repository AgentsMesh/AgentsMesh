package gitprovider

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
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
	HttpCloneURL    string `gorm:"size:500" json:"http_clone_url"`             // HTTPS clone URL
	SshCloneURL     string `gorm:"size:500" json:"ssh_clone_url"`              // SSH clone URL

	ExternalID    string  `gorm:"size:255;not null" json:"external_id"`
	Name          string  `gorm:"size:255;not null" json:"name"`
	Slug          string  `gorm:"column:slug;size:500;not null" json:"slug"`
	DefaultBranch string  `gorm:"size:100;default:'main'" json:"default_branch"`
	TicketPrefix  *string `gorm:"size:10" json:"ticket_prefix,omitempty"`

	// Visibility: "organization" (all members can see), "private" (only importer can see)
	Visibility       string `gorm:"size:20;not null;default:'organization'" json:"visibility"`
	ImportedByUserID *int64 `gorm:"index" json:"imported_by_user_id,omitempty"` // User who imported this repo

	// Workspace preparation
	PreparationScript  *string `gorm:"type:text" json:"preparation_script,omitempty"`    // Script to run after worktree creation
	PreparationTimeout *int    `gorm:"default:300" json:"preparation_timeout,omitempty"` // Script timeout in seconds (default 300)

	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	// Webhook configuration stored as JSONB
	WebhookConfig *WebhookConfig `gorm:"type:jsonb" json:"webhook_config,omitempty"`

	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"` // Soft delete support
}

func (Repository) TableName() string {
	return "repositories"
}

// WebhookConfig represents webhook configuration for a repository
type WebhookConfig struct {
	ID               string   `json:"id"`
	URL              string   `json:"url"`
	Secret           string   `json:"secret,omitempty"`     // Repository-specific webhook secret (not exposed in API responses)
	Events           []string `json:"events"`
	IsActive         bool     `json:"is_active"`
	NeedsManualSetup bool     `json:"needs_manual_setup"`   // Whether manual configuration is required
	LastError        string   `json:"last_error,omitempty"` // Last error message
	CreatedAt        string   `json:"created_at,omitempty"`
}

// Value implements driver.Valuer for GORM JSONB support
func (wc WebhookConfig) Value() (driver.Value, error) {
	return json.Marshal(wc)
}

// Scan implements sql.Scanner for GORM JSONB support
func (wc *WebhookConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal WebhookConfig: value is not []byte")
	}
	if len(bytes) == 0 {
		return nil
	}
	return json.Unmarshal(bytes, wc)
}

// WebhookStatus represents the public-facing webhook status (without secret)
type WebhookStatus struct {
	Registered   bool     `json:"registered"`
	WebhookID    string   `json:"webhook_id,omitempty"`
	WebhookURL   string   `json:"webhook_url,omitempty"`
	Events       []string `json:"events,omitempty"`
	IsActive     bool     `json:"is_active"`
	NeedsManualSetup bool `json:"needs_manual_setup"`
	LastError    string   `json:"last_error,omitempty"`
	RegisteredAt string   `json:"registered_at,omitempty"`
}

// ToStatus converts WebhookConfig to WebhookStatus (hiding the secret)
func (wc *WebhookConfig) ToStatus() *WebhookStatus {
	if wc == nil {
		return &WebhookStatus{Registered: false}
	}
	return &WebhookStatus{
		Registered:       wc.ID != "" || wc.NeedsManualSetup,
		WebhookID:        wc.ID,
		WebhookURL:       wc.URL,
		Events:           wc.Events,
		IsActive:         wc.IsActive,
		NeedsManualSetup: wc.NeedsManualSetup,
		LastError:        wc.LastError,
		RegisteredAt:     wc.CreatedAt,
	}
}
