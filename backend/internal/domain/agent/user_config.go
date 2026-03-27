package agent

import (
	"time"
)

// UserAgentConfig represents user-level personal agent configuration
type UserAgentConfig struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	UserID    int64  `gorm:"not null;index" json:"user_id"`
	AgentSlug string `gorm:"size:100;not null;index;column:agent_slug" json:"agent_slug"`

	ConfigValues ConfigValues `gorm:"type:jsonb;not null;default:'{}'" json:"config_values"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	Agent *Agent `gorm:"foreignKey:AgentSlug;references:Slug" json:"agent,omitempty"`
}

func (UserAgentConfig) TableName() string {
	return "user_agent_configs"
}

// UserAgentConfigResponse is the API response for user agent config
type UserAgentConfigResponse struct {
	ID            int64                  `json:"id"`
	UserID        int64                  `json:"user_id"`
	AgentSlug     string                 `json:"agent_slug"`
	AgentName string                 `json:"agent_name,omitempty"`
	ConfigValues  map[string]interface{} `json:"config_values"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

// ToResponse converts UserAgentConfig to API response
func (c *UserAgentConfig) ToResponse() *UserAgentConfigResponse {
	resp := &UserAgentConfigResponse{
		ID:           c.ID,
		UserID:       c.UserID,
		AgentSlug:    c.AgentSlug,
		ConfigValues: c.ConfigValues,
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
	}

	if c.Agent != nil {
		resp.AgentName = c.Agent.Name
	}

	return resp
}
