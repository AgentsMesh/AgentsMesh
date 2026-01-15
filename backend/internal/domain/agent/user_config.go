package agent

import (
	"time"
)

// UserAgentConfig represents user-level personal agent configuration
// This replaces organization-level config for a more user-centric approach
type UserAgentConfig struct {
	ID          int64 `gorm:"primaryKey" json:"id"`
	UserID      int64 `gorm:"not null;index" json:"user_id"`
	AgentTypeID int64 `gorm:"not null;index" json:"agent_type_id"`

	// Dynamic configuration values (JSON)
	ConfigValues ConfigValues `gorm:"type:jsonb;not null;default:'{}'" json:"config_values"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	AgentType *AgentType `gorm:"foreignKey:AgentTypeID" json:"agent_type,omitempty"`
}

func (UserAgentConfig) TableName() string {
	return "user_agent_configs"
}

// UserAgentConfigResponse is the API response for user agent config
type UserAgentConfigResponse struct {
	ID            int64                  `json:"id"`
	UserID        int64                  `json:"user_id"`
	AgentTypeID   int64                  `json:"agent_type_id"`
	AgentTypeName string                 `json:"agent_type_name,omitempty"`
	AgentTypeSlug string                 `json:"agent_type_slug,omitempty"`
	ConfigValues  map[string]interface{} `json:"config_values"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

// ToResponse converts UserAgentConfig to API response
func (c *UserAgentConfig) ToResponse() *UserAgentConfigResponse {
	resp := &UserAgentConfigResponse{
		ID:           c.ID,
		UserID:       c.UserID,
		AgentTypeID:  c.AgentTypeID,
		ConfigValues: c.ConfigValues,
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
	}

	if c.AgentType != nil {
		resp.AgentTypeName = c.AgentType.Name
		resp.AgentTypeSlug = c.AgentType.Slug
	}

	return resp
}
