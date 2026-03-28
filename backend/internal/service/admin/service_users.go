package admin

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

// UserListQuery represents query parameters for user listing
type UserListQuery struct {
	Search   string
	IsActive *bool
	IsAdmin  *bool
	Page     int
	PageSize int
}

// UserListResponse represents paginated user list response
type UserListResponse struct {
	Data       []user.User `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// ListUsers retrieves users with filtering and pagination
func (s *Service) ListUsers(ctx context.Context, query *UserListQuery) (*UserListResponse, error) {
	db := s.db.Model(&user.User{})

	// Apply filters
	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("email ILIKE ? OR username ILIKE ? OR name ILIKE ?", searchPattern, searchPattern, searchPattern)
	}
	if query.IsActive != nil {
		db = db.Where("is_active = ?", *query.IsActive)
	}
	if query.IsAdmin != nil {
		db = db.Where("is_system_admin = ?", *query.IsAdmin)
	}

	// Count total
	var total int64
	if err := db.Count(&total); err != nil {
		return nil, err
	}

	// Apply pagination using helper
	p := normalizePagination(query.Page, query.PageSize, total)

	var users []user.User
	if err := db.
		Order("created_at DESC").
		Limit(p.PageSize).
		Offset(p.Offset).
		Find(&users); err != nil {
		return nil, err
	}

	return &UserListResponse{
		Data:       users,
		Total:      total,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: p.TotalPages,
	}, nil
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, userID int64) (*user.User, error) {
	var u user.User
	if err := s.db.First(&u, userID); err != nil {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

// UpdateUser updates a user's profile
func (s *Service) UpdateUser(ctx context.Context, userID int64, updates map[string]interface{}) (*user.User, error) {
	var u user.User
	if err := s.db.First(&u, userID); err != nil {
		return nil, ErrUserNotFound
	}

	if err := s.db.Updates(&u, updates); err != nil {
		slog.Error("admin: failed to update user", "user_id", userID, "error", err)
		return nil, err
	}

	slog.Info("admin: user updated", "user_id", userID)

	// Reload
	if err := s.db.First(&u, userID); err != nil {
		return nil, err
	}
	return &u, nil
}

// DisableUser disables a user account
func (s *Service) DisableUser(ctx context.Context, userID int64) (*user.User, error) {
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_active": false})
}

// EnableUser enables a user account
func (s *Service) EnableUser(ctx context.Context, userID int64) (*user.User, error) {
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_active": true})
}

// GrantAdmin grants system admin privileges to a user
func (s *Service) GrantAdmin(ctx context.Context, userID int64) (*user.User, error) {
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_system_admin": true})
}

// RevokeAdmin revokes system admin privileges from a user
func (s *Service) RevokeAdmin(ctx context.Context, userID int64, currentAdminID int64) (*user.User, error) {
	if userID == currentAdminID {
		return nil, ErrCannotRevokeOwnAdmin
	}
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_system_admin": false})
}

// VerifyUserEmail marks a user's email as verified
func (s *Service) VerifyUserEmail(ctx context.Context, userID int64) (*user.User, error) {
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_email_verified": true})
}

// UnverifyUserEmail marks a user's email as unverified
func (s *Service) UnverifyUserEmail(ctx context.Context, userID int64) (*user.User, error) {
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_email_verified": false})
}
