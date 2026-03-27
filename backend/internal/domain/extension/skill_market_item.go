package extension

import (
	"encoding/json"
	"time"
)

// SkillMarketItem represents a Skill available in the marketplace
type SkillMarketItem struct {
	ID              int64           `gorm:"primaryKey" json:"id"`
	RegistryID      int64           `gorm:"column:registry_id;not null" json:"registry_id"`
	Slug            string          `gorm:"size:100;not null" json:"slug"`
	DisplayName     string          `gorm:"size:100" json:"display_name,omitempty"`
	Description     string          `gorm:"size:1024" json:"description,omitempty"`
	License         string          `gorm:"size:100" json:"license,omitempty"`
	Compatibility   string          `gorm:"size:500" json:"compatibility,omitempty"`
	AllowedTools    string          `json:"allowed_tools,omitempty"`
	Metadata        json.RawMessage `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	Category        string          `gorm:"size:50" json:"category,omitempty"`
	ContentSha      string          `gorm:"size:64;not null" json:"content_sha"`
	StorageKey      string          `gorm:"size:500;not null" json:"storage_key"`
	PackageSize     int64           `json:"package_size"`
	Version         int             `gorm:"default:1" json:"version"`
	AgentFilter json.RawMessage `gorm:"type:jsonb;default:'[\"claude-code\"]'" json:"agent_filter,omitempty"`
	IsActive        bool            `gorm:"not null;default:true" json:"is_active"`
	CreatedAt       time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	Registry *SkillRegistry `gorm:"foreignKey:RegistryID" json:"registry,omitempty"`
}

func (SkillMarketItem) TableName() string { return "skill_market_items" }

// GetAgentFilter parses and returns the agent_filter as a string slice.
// Returns nil if the filter is empty or null (meaning all agents are allowed).
func (m *SkillMarketItem) GetAgentFilter() []string {
	if len(m.AgentFilter) == 0 {
		return nil
	}
	var filter []string
	if err := json.Unmarshal(m.AgentFilter, &filter); err != nil {
		return nil
	}
	return filter
}
