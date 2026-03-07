package invitation

import (
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/invitation"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
)

var (
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExpired  = errors.New("invitation has expired")
	ErrInvitationAccepted = errors.New("invitation already accepted")
	ErrAlreadyMember      = errors.New("user is already a member of this organization")
	ErrPendingInvitation  = errors.New("a pending invitation already exists for this email")
	ErrInvalidRole        = errors.New("invalid role")
	ErrNotAuthorized      = errors.New("not authorized to manage invitations")
)

const (
	// InvitationValidDays is the number of days an invitation is valid
	InvitationValidDays = 7
)

// Service handles invitation operations
type Service struct {
	repo         invitation.Repository
	emailService email.Service
}

// NewService creates a new invitation service
func NewService(repo invitation.Repository, emailService email.Service) *Service {
	return &Service{
		repo:         repo,
		emailService: emailService,
	}
}
