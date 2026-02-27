package extension

import "time"

// SkillRegistryOverride allows an organization to disable a platform-level skill registry
type SkillRegistryOverride struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	OrganizationID int64     `gorm:"column:organization_id;uniqueIndex:idx_registry_override_unique" json:"organization_id"`
	RegistryID     int64     `gorm:"column:registry_id;uniqueIndex:idx_registry_override_unique" json:"registry_id"`
	IsDisabled     bool      `json:"is_disabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (SkillRegistryOverride) TableName() string {
	return "skill_registry_overrides"
}
