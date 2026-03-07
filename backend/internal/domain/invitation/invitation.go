package invitation

import (
	"context"
	"time"
)

// Invitation represents a pending organization member invitation
type Invitation struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	OrganizationID int64      `gorm:"not null;index" json:"organization_id"`
	Email          string     `gorm:"size:255;not null;index" json:"email"`
	Role           string     `gorm:"size:20;not null;default:member" json:"role"`
	Token          string     `gorm:"size:255;not null;uniqueIndex" json:"-"`
	InvitedBy      int64      `gorm:"not null" json:"invited_by"`
	ExpiresAt      time.Time  `gorm:"not null" json:"expires_at"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Invitation) TableName() string {
	return "invitations"
}

// IsExpired checks if the invitation has expired
func (i *Invitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsAccepted checks if the invitation has been accepted
func (i *Invitation) IsAccepted() bool {
	return i.AcceptedAt != nil
}

// IsPending checks if the invitation is still pending
func (i *Invitation) IsPending() bool {
	return !i.IsAccepted() && !i.IsExpired()
}

// AcceptInvitationParams holds parameters for the atomic accept operation
type AcceptInvitationParams struct {
	Invitation *Invitation
	UserID     int64
	Role       string
}

// AcceptInvitationResult holds the result of accepting an invitation
type AcceptInvitationResult struct {
	OrganizationID int64
	MemberID       int64
}

// OrgInfo holds basic organization information
type OrgInfo struct {
	Name string
	Slug string
}

// Repository defines the interface for invitation data access
type Repository interface {
	Create(ctx context.Context, invitation *Invitation) error
	GetByToken(ctx context.Context, token string) (*Invitation, error)
	GetByID(ctx context.Context, id int64) (*Invitation, error)
	GetByOrgAndEmail(ctx context.Context, orgID int64, email string) (*Invitation, error)
	ListByOrganization(ctx context.Context, orgID int64) ([]*Invitation, error)
	ListPendingByEmail(ctx context.Context, email string) ([]*Invitation, error)
	Update(ctx context.Context, invitation *Invitation) error
	Delete(ctx context.Context, id int64) error
	DeleteExpired(ctx context.Context) error

	// Cross-entity queries (avoid Service needing direct DB access)
	CheckMemberExists(ctx context.Context, orgID int64, userID int64) (bool, error)
	CheckMemberExistsByEmail(ctx context.Context, orgID int64, email string) (bool, error)
	GetOrganization(ctx context.Context, orgID int64) (*OrgInfo, error)
	GetUserDisplayName(ctx context.Context, userID int64) (string, error)

	// AcceptInvitationAtomic atomically adds a member and marks the invitation as accepted
	AcceptInvitationAtomic(ctx context.Context, params *AcceptInvitationParams) (*AcceptInvitationResult, error)
}
