package grant

import (
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

const (
	TypePod        = "pod"
	TypeRunner     = "runner"
	TypeRepository = "repository"
)

func IntResourceID(id int64) string {
	return strconv.FormatInt(id, 10)
}

type ResourceGrant struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	OrganizationID int64      `gorm:"not null;index" json:"organization_id"`
	ResourceType   string     `gorm:"size:32;not null" json:"resource_type"`
	ResourceID     string     `gorm:"size:64;not null" json:"resource_id"`
	UserID         int64      `gorm:"not null" json:"user_id"`
	GrantedBy      int64      `gorm:"not null" json:"granted_by"`
	CreatedAt      time.Time  `gorm:"not null;default:now()" json:"created_at"`
	User           *user.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	GrantedByUser  *user.User `gorm:"foreignKey:GrantedBy" json:"granted_by_user,omitempty"`
}

func (ResourceGrant) TableName() string { return "resource_grants" }
