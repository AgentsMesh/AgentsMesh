package extension

import (
	"encoding/json"
	"time"
)

// Sync status constants
const (
	SyncStatusPending = "pending"
	SyncStatusSyncing = "syncing"
	SyncStatusSuccess = "success"
	SyncStatusFailed  = "failed"
)

// Source type constants
const (
	SourceTypeAuto       = "auto"
	SourceTypeCollection = "collection"
	SourceTypeSingle     = "single"
)

// Auth type constants
const (
	AuthTypeNone      = "none"
	AuthTypeGitHubPAT = "github_pat"
	AuthTypeGitLabPAT = "gitlab_pat"
	AuthTypeSSHKey    = "ssh_key"
)

// SkillRegistry represents a GitHub/GitLab repository used as a Skill import source
type SkillRegistry struct {
	ID               int64           `gorm:"primaryKey" json:"id"`
	OrganizationID   *int64          `gorm:"index" json:"organization_id"`                          // NULL = platform-level
	RepositoryURL    string          `gorm:"size:500;not null" json:"repository_url"`
	Branch           string          `gorm:"size:100;default:main" json:"branch"`
	SourceType       string          `gorm:"size:20;default:auto" json:"source_type"`               // auto / collection / single
	DetectedType     string          `gorm:"size:20" json:"detected_type,omitempty"`                // collection / single
	CompatibleAgents json.RawMessage `gorm:"type:jsonb;default:'[\"claude-code\"]'" json:"compatible_agents,omitempty"` // agent whitelist
	AuthType         string          `gorm:"size:20;default:none" json:"auth_type"`                 // none / github_pat / gitlab_pat / ssh_key
	AuthCredential   string          `gorm:"column:auth_credential" json:"-"`                       // encrypted, never exposed in JSON
	LastSyncedAt     *time.Time      `json:"last_synced_at,omitempty"`
	LastCommitSha    string          `gorm:"size:40" json:"last_commit_sha,omitempty"`
	SyncStatus       string          `gorm:"size:20;default:pending" json:"sync_status"`            // pending/syncing/success/failed
	SyncError        string          `json:"sync_error,omitempty"`
	SkillCount       int             `gorm:"default:0" json:"skill_count"`
	IsActive         bool            `gorm:"not null;default:true" json:"is_active"`
	CreatedAt        time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt        time.Time       `gorm:"not null;default:now()" json:"updated_at"`
}

func (SkillRegistry) TableName() string { return "skill_registries" }

// IsPlatformLevel returns true if this is a platform-level registry (not org-specific)
func (s *SkillRegistry) IsPlatformLevel() bool {
	return s.OrganizationID == nil
}

// GetCompatibleAgents parses and returns the compatible_agents as a string slice.
// Returns nil if the field is empty or null (meaning all agents are allowed).
func (s *SkillRegistry) GetCompatibleAgents() []string {
	if len(s.CompatibleAgents) == 0 {
		return nil
	}
	var agents []string
	if err := json.Unmarshal(s.CompatibleAgents, &agents); err != nil {
		return nil
	}
	return agents
}

// HasAuth returns true if this registry has authentication configured
func (s *SkillRegistry) HasAuth() bool {
	return s.AuthType != "" && s.AuthType != AuthTypeNone
}

// HasAuthConfigured returns true if auth_type indicates a credential is expected.
// The actual credential string is checked separately to avoid exposing it.
func (s *SkillRegistry) HasAuthConfigured() bool {
	return s.HasAuth() && s.AuthCredential != ""
}
