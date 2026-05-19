package agent

import (
	"time"

	"github.com/anthropics/agentsmesh/agentfile/extract"
	"github.com/anthropics/agentsmesh/agentfile/parser"
)

const RunnerHostProfileID int64 = 0

type UserAgentCredentialProfile struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	UserID    int64  `gorm:"not null;index" json:"user_id"`
	AgentSlug string `gorm:"size:100;not null;index;column:agent_slug" json:"agent_slug"`

	Name        string  `gorm:"size:100;not null" json:"name"`
	Description *string `gorm:"type:text" json:"description,omitempty"`

	IsRunnerHost bool `gorm:"not null;default:false" json:"is_runner_host"`

	CredentialsEncrypted EncryptedCredentials `gorm:"type:jsonb" json:"-"`

	IsDefault bool `gorm:"not null;default:false" json:"is_default"`
	IsActive  bool `gorm:"not null;default:true" json:"is_active"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	Agent *Agent `gorm:"foreignKey:AgentSlug;references:Slug" json:"agent,omitempty"`
}

func (UserAgentCredentialProfile) TableName() string {
	return "user_agent_credential_profiles"
}

type CredentialProfileResponse struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	AgentSlug string `json:"agent_slug"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	IsRunnerHost bool `json:"is_runner_host"`
	IsDefault    bool `json:"is_default"`
	IsActive     bool `json:"is_active"`

	ConfiguredFields []string `json:"configured_fields,omitempty"`

	ConfiguredValues map[string]string `json:"configured_values,omitempty"`

	AgentName string `json:"agent_name,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

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

	fieldTypes := make(map[string]string)
	if p.Agent != nil {
		fieldTypes = extractCredentialFieldTypes(p.Agent.AgentfileSource)
	}

	if p.CredentialsEncrypted != nil {
		fields := make([]string, 0, len(p.CredentialsEncrypted))
		values := make(map[string]string)

		for k, v := range p.CredentialsEncrypted {
			fields = append(fields, k)
			if fieldTypes[k] == "text" && v != "" {
				values[k] = v
			}
		}

		resp.ConfiguredFields = fields
		if len(values) > 0 {
			resp.ConfiguredValues = values
		}
	}

	if p.Agent != nil {
		resp.AgentName = p.Agent.Name
	}

	return resp
}

type CredentialProfilesByAgent struct {
	AgentSlug string                       `json:"agent_slug"`
	AgentName string                       `json:"agent_name"`
	Profiles  []*CredentialProfileResponse `json:"profiles"`
}

type ListCredentialProfilesResponse struct {
	Items []*CredentialProfilesByAgent `json:"items"`
}

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
