package admin

import (
	"context"
	"log/slog"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

type UserListQuery struct {
	Search   string
	IsActive *bool
	IsAdmin  *bool
	Page     int
	PageSize int
}

type UserListResponse struct {
	Data       []user.User `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

func (s *Service) ListUsers(ctx context.Context, query *UserListQuery) (*UserListResponse, error) {
	db := s.db.Model(&user.User{})

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

	var total int64
	if err := db.Count(&total); err != nil {
		return nil, err
	}

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

func (s *Service) GetUser(ctx context.Context, userID int64) (*user.User, error) {
	var u user.User
	if err := s.db.First(&u, userID); err != nil {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

func (s *Service) UpdateUser(ctx context.Context, userID int64, updates map[string]interface{}) (*user.User, error) {
	var u user.User
	if err := s.db.First(&u, userID); err != nil {
		return nil, ErrUserNotFound
	}

	if err := s.db.Updates(&u, updates); err != nil {
		if isUniqueViolation(err) {
			if _, ok := updates["username"]; ok {
				return nil, ErrUsernameAlreadyExists
			}
			if _, ok := updates["email"]; ok {
				return nil, ErrEmailAlreadyExists
			}
		}
		slog.ErrorContext(ctx, "admin: failed to update user", "user_id", userID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "admin: user updated", "user_id", userID)

	if err := s.db.First(&u, userID); err != nil {
		return nil, err
	}
	return &u, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLSTATE 23505") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "UNIQUE constraint failed")
}

func (s *Service) DisableUser(ctx context.Context, userID int64) (*user.User, error) {
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_active": false})
}

func (s *Service) EnableUser(ctx context.Context, userID int64) (*user.User, error) {
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_active": true})
}

func (s *Service) GrantAdmin(ctx context.Context, userID int64) (*user.User, error) {
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_system_admin": true})
}

func (s *Service) RevokeAdmin(ctx context.Context, userID int64, currentAdminID int64) (*user.User, error) {
	if userID == currentAdminID {
		return nil, ErrCannotRevokeOwnAdmin
	}
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_system_admin": false})
}

func (s *Service) VerifyUserEmail(ctx context.Context, userID int64) (*user.User, error) {
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_email_verified": true})
}

func (s *Service) UnverifyUserEmail(ctx context.Context, userID int64) (*user.User, error) {
	return s.UpdateUser(ctx, userID, map[string]interface{}{"is_email_verified": false})
}
