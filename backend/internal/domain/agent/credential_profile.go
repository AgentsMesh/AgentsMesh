package agent

import (
	"time"
)

// RunnerHostProfileID is a special constant indicating "RunnerHost" mode
// When credential_profile_id is 0 or nil, it means using Runner's local environment
// and no credentials will be injected into the pod
const RunnerHostProfileID int64 = 0

// UserAgentCredentialProfile represents a user's credential configuration profile for an agent type
// Each user can have multiple profiles per agent type (e.g., RunnerHost, work config, proxy config)
type UserAgentCredentialProfile struct {
	ID          int64 `gorm:"primaryKey" json:"id"`
	UserID      int64 `gorm:"not null;index" json:"user_id"`
	AgentTypeID int64 `gorm:"not null;index" json:"agent_type_id"`

	// Profile info
	Name        string  `gorm:"size:100;not null" json:"name"`
	Description *string `gorm:"type:text" json:"description,omitempty"`

	// Credential type: true = use Runner's local environment, no credentials injected
	IsRunnerHost bool `gorm:"not null;default:false" json:"is_runner_host"`

	// Encrypted credentials (only used when is_runner_host = false)
	// Stored as: {"base_url": "xxx", "api_key": "xxx", ...}
	CredentialsEncrypted EncryptedCredentials `gorm:"type:jsonb" json:"-"`

	// Status flags
	IsDefault bool `gorm:"not null;default:false" json:"is_default"`
	IsActive  bool `gorm:"not null;default:true" json:"is_active"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	AgentType *AgentType `gorm:"foreignKey:AgentTypeID" json:"agent_type,omitempty"`
}

func (UserAgentCredentialProfile) TableName() string {
	return "user_agent_credential_profiles"
}

// CredentialProfileResponse is the API response for credential profile
type CredentialProfileResponse struct {
	ID          int64   `json:"id"`
	UserID      int64   `json:"user_id"`
	AgentTypeID int64   `json:"agent_type_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	IsRunnerHost bool `json:"is_runner_host"`
	IsDefault    bool `json:"is_default"`
	IsActive     bool `json:"is_active"`

	// Show which fields have been configured (without exposing actual values)
	ConfiguredFields []string `json:"configured_fields,omitempty"`

	// AgentType info
	AgentTypeName string `json:"agent_type_name,omitempty"`
	AgentTypeSlug string `json:"agent_type_slug,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ToResponse converts UserAgentCredentialProfile to API response
func (p *UserAgentCredentialProfile) ToResponse() *CredentialProfileResponse {
	resp := &CredentialProfileResponse{
		ID:           p.ID,
		UserID:       p.UserID,
		AgentTypeID:  p.AgentTypeID,
		Name:         p.Name,
		Description:  p.Description,
		IsRunnerHost: p.IsRunnerHost,
		IsDefault:    p.IsDefault,
		IsActive:     p.IsActive,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    p.UpdatedAt.Format(time.RFC3339),
	}

	// Extract configured field names (without exposing values)
	if p.CredentialsEncrypted != nil {
		fields := make([]string, 0, len(p.CredentialsEncrypted))
		for k := range p.CredentialsEncrypted {
			fields = append(fields, k)
		}
		resp.ConfiguredFields = fields
	}

	// AgentType info
	if p.AgentType != nil {
		resp.AgentTypeName = p.AgentType.Name
		resp.AgentTypeSlug = p.AgentType.Slug
	}

	return resp
}

// CredentialProfilesByAgentType groups profiles by agent type for list response
type CredentialProfilesByAgentType struct {
	AgentTypeID   int64                        `json:"agent_type_id"`
	AgentTypeName string                       `json:"agent_type_name"`
	AgentTypeSlug string                       `json:"agent_type_slug"`
	Profiles      []*CredentialProfileResponse `json:"profiles"`
}

// ListCredentialProfilesResponse is the response for listing all user credential profiles
type ListCredentialProfilesResponse struct {
	Items []*CredentialProfilesByAgentType `json:"items"`
}
