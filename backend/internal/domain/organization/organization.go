package organization

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

// Organization role constants
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Organization represents a tenant in the multi-tenant system
type Organization struct {
	ID   int64  `gorm:"primaryKey" json:"id"`
	Name string `gorm:"size:100;not null" json:"name"`
	Slug string `gorm:"size:100;not null;uniqueIndex" json:"slug"`

	LogoURL *string `gorm:"type:text" json:"logo_url,omitempty"`

	// Subscription info
	SubscriptionPlan   string `gorm:"size:50;not null;default:'based'" json:"subscription_plan"`
	SubscriptionStatus string `gorm:"size:20;not null;default:'trialing'" json:"subscription_status"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Role is populated by ListByUser to indicate the requesting user's role.
	// Not a DB column — filled via raw scan in the repo query.
	Role string `gorm:"->" json:"role,omitempty"`

	// Associations
	Members []Member `gorm:"foreignKey:OrganizationID" json:"members,omitempty"`
}

func (Organization) TableName() string {
	return "organizations"
}

// GetID returns the organization ID (implements OrganizationGetter interface)
func (o *Organization) GetID() int64 {
	return o.ID
}

// GetSlug returns the organization slug (implements OrganizationGetter interface)
func (o *Organization) GetSlug() string {
	return o.Slug
}

// GetName returns the organization name (implements OrganizationGetter interface)
func (o *Organization) GetName() string {
	return o.Name
}

// Member represents an organization membership
type Member struct {
	ID             int64  `gorm:"primaryKey" json:"id"`
	OrganizationID int64  `gorm:"not null;index" json:"organization_id"`
	UserID         int64  `gorm:"not null;index" json:"user_id"`
	Role           string `gorm:"size:50;not null;default:'member'" json:"role"` // owner, admin, member

	JoinedAt time.Time `gorm:"not null;default:now()" json:"joined_at"`

	// Associations
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	User         *user.User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Member) TableName() string {
	return "organization_members"
}
