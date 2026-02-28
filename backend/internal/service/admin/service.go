package admin

import (
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
)

var (
	ErrUserNotFound                = errors.New("user not found")
	ErrOrganizationNotFound        = errors.New("organization not found")
	ErrSubscriptionNotFound        = errors.New("subscription not found")
	ErrRunnerNotFound              = errors.New("runner not found")
	ErrPromoCodeNotFound           = errors.New("promo code not found")
	ErrPromoCodeAlreadyExists      = errors.New("promo code already exists")
	ErrPromoCodeHasRedemptions     = errors.New("promo code has redemptions")
	ErrCannotRevokeOwnAdmin        = errors.New("cannot revoke your own admin privileges")
	ErrCannotDisableSelf           = errors.New("cannot disable your own account")
	ErrOrganizationHasActiveRunner = errors.New("cannot delete organization with active runners")
	ErrRunnerHasActivePods         = errors.New("cannot delete runner with active pods")
	ErrRunnerHasLoopRefs           = errors.New("cannot delete runner referenced by one or more loops")
)

// Service provides admin management functionality
type Service struct {
	db database.DB
}

// NewService creates a new admin service
func NewService(db database.DB) *Service {
	return &Service{db: db}
}

// PaginationParams holds normalized pagination parameters
type PaginationParams struct {
	Page       int
	PageSize   int
	Offset     int
	TotalPages int
}

// normalizePagination normalizes and calculates pagination parameters
func normalizePagination(page, pageSize int, total int64) PaginationParams {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// Safe calculation to avoid overflow: (total + pageSize - 1) / pageSize
	totalPages := (int(total) + pageSize - 1) / pageSize
	if totalPages < 0 {
		totalPages = 0
	}

	return PaginationParams{
		Page:       page,
		PageSize:   pageSize,
		Offset:     offset,
		TotalPages: totalPages,
	}
}
