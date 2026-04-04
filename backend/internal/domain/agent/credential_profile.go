package agent

import (
	"time"

	"github.com/anthropics/agentsmesh/agentfile/extract"
	"github.com/anthropics/agentsmesh/agentfile/parser"
)

// RunnerHostProfileID is a special sentinel value indicating explicit "RunnerHost" mode.
// When credential_profile_id is explicitly set to 0, the system uses the Runner's local
// environment and no credentials are injected into the pod.
// This is distinct from nil/absent, which means "use user's default profile".
const RunnerHostProfileID int64 = 0

// UserAgentCredentialProfile represents a user's credential configuration profile for an agent
// Each user can have multiple profiles per agent (e.g., RunnerHost, work config, proxy config)
type UserAgentCredentialProfile struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	UserID    int64  `gorm:"not null;index" json:"user_id"`
	AgentSlug string `gorm:"size:100;not null;index;column:agent_slug" json:"agent_slug"`

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
	Agent *Agent `gorm:"foreignKey:AgentSlug;references:Slug" json:"agent,omitempty"`
}

func (UserAgentCredentialProfile) TableName() string {
	return "user_agent_credential_profiles"
}

// CredentialProfileResponse is the API response for credential profile
type CredentialProfileResponse struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	AgentSlug string `json:"agent_slug"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	IsRunnerHost bool `json:"is_runner_host"`
	IsDefault    bool `json:"is_default"`
	IsActive     bool `json:"is_active"`

	// Show which fields have been configured (without exposing actual values)
	ConfiguredFields []string `json:"configured_fields,omitempty"`

	// Non-secret field values that can be echoed back for editing (e.g. base_url)
	ConfiguredValues map[string]string `json:"configured_values,omitempty"`

	// Agent info
	AgentName string `json:"agent_name,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ToResponse converts UserAgentCredentialProfile to API response.
// Non-secret credential values (type: "text") are included in ConfiguredValues
// so the frontend can echo them back during editing. Secret values are only
// listed in ConfiguredFields (name only, no value).
func (p *UserAgentCredentialProfile) ToResponse() *CredentialProfileResponse {
	resp := &CredentialProfileResponse{
		ID:           p.ID,
		UserID:       p.UserID,
		AgentSlug:    p.AgentSlug,
		Name:         p.Name,
		Description:  p.Description,
		IsRunnerHost: p.IsRunnerHost,
		IsDefault:    p.IsDefault,
		IsActive:     p.IsActive,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    p.UpdatedAt.Format(time.RFC3339),
	}

	// Build a lookup of field type from AgentFile ENV declarations (requires Agent preloaded)
	fieldTypes := make(map[string]string)
	if p.Agent != nil {
		fieldTypes = extractCredentialFieldTypes(p.Agent.AgentfileSource)
	}

	// Separate credentials into ConfiguredFields (names only) and ConfiguredValues (non-secret values)
	if p.CredentialsEncrypted != nil {
		fields := make([]string, 0, len(p.CredentialsEncrypted))
		values := make(map[string]string)

		for k, v := range p.CredentialsEncrypted {
			fields = append(fields, k)
			// Only expose non-secret values (type: "text") for edit echoing
			if fieldTypes[k] == "text" && v != "" {
				values[k] = v
			}
		}

		resp.ConfiguredFields = fields
		if len(values) > 0 {
			resp.ConfiguredValues = values
		}
	}

	// Agent info
	if p.Agent != nil {
		resp.AgentName = p.Agent.Name
	}

	return resp
}

// CredentialProfilesByAgent groups profiles by agent for list response
type CredentialProfilesByAgent struct {
	AgentSlug string                       `json:"agent_slug"`
	AgentName string                       `json:"agent_name"`
	Profiles  []*CredentialProfileResponse `json:"profiles"`
}

// ListCredentialProfilesResponse is the response for listing all user credential profiles
type ListCredentialProfilesResponse struct {
	Items []*CredentialProfilesByAgent `json:"items"`
}

// extractCredentialFieldTypes extracts ENV field types from AgentFile source.
// Returns a map of field name -> source type ("secret" or "text").
func extractCredentialFieldTypes(agentfileSource *string) map[string]string {
	types := make(map[string]string)
	if agentfileSource == nil || *agentfileSource == "" {
		return types
	}
	prog, errs := parser.Parse(*agentfileSource)
	if len(errs) > 0 || prog == nil {
		return types
	}
	spec := extract.Extract(prog)
	for _, env := range spec.Env {
		if env.Source != "" {
			types[env.Name] = env.Source
		}
	}
	return types
}
