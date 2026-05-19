package invitation

import (
	"context"
	"time"
)

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

func (Invitation) TableName() string {
	return "invitations"
}

func (i *Invitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

func (i *Invitation) IsAccepted() bool {
	return i.AcceptedAt != nil
}

func (i *Invitation) IsPending() bool {
	return !i.IsAccepted() && !i.IsExpired()
}

type AcceptInvitationParams struct {
	Invitation *Invitation
	UserID     int64
	Role       string
}

type AcceptInvitationResult struct {
	OrganizationID int64
	MemberID       int64
}

type OrgInfo struct {
	Name string
	Slug string
}

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

	CheckMemberExists(ctx context.Context, orgID int64, userID int64) (bool, error)
	CheckMemberExistsByEmail(ctx context.Context, orgID int64, email string) (bool, error)
	GetOrganization(ctx context.Context, orgID int64) (*OrgInfo, error)
	GetUserDisplayName(ctx context.Context, userID int64) (string, error)

	// AcceptInvitationAtomic atomically adds a member and marks the invitation as accepted
	AcceptInvitationAtomic(ctx context.Context, params *AcceptInvitationParams) (*AcceptInvitationResult, error)
}
