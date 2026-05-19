package auth

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
)

func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	if s.ssoChecker != nil && strings.Contains(email, "@") {
		isSystemAdmin := false
		u, err := s.userService.GetByEmail(ctx, email)
		if err == nil && u != nil {
			isSystemAdmin = u.IsSystemAdmin
		}

		allowed, err := s.ssoChecker.IsPasswordLoginAllowed(ctx, email, isSystemAdmin)
		if err == nil && !allowed {
			return nil, ErrSSOEnforced
		}
	}

	u, err := s.userService.Authenticate(ctx, email, password)
	if err != nil {
		if errors.Is(err, userService.ErrInvalidCredentials) {
			slog.WarnContext(ctx, "login failed", "email", email, "reason", "invalid_credentials")
			return nil, ErrInvalidCredentials
		}
		if errors.Is(err, userService.ErrUserInactive) {
			slog.WarnContext(ctx, "login failed", "email", email, "reason", "user_disabled")
			return nil, ErrUserDisabled
		}
		slog.WarnContext(ctx, "login failed", "email", email, "reason", "internal_error")
		return nil, err
	}

	tokens, err := s.GenerateTokenPair(u, 0, "")
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "user logged in", "user_id", u.ID, "email", email)

	return &LoginResult{
		User:         u,
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
	}, nil
}

func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*LoginResult, error) {
	if err := domainUser.ValidateUsername(req.Username); err != nil {
		return nil, err
	}

	if s.ssoChecker != nil && strings.Contains(req.Email, "@") {
		allowed, err := s.ssoChecker.IsPasswordLoginAllowed(ctx, req.Email, false)
		if err == nil && !allowed {
			return nil, ErrSSOEnforced
		}
	}

	u, err := s.userService.Create(ctx, &userService.CreateRequest{
		Email:    req.Email,
		Username: req.Username,
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, userService.ErrEmailAlreadyExists) {
			return nil, ErrEmailExists
		}
		if errors.Is(err, userService.ErrUsernameExists) {
			return nil, ErrUsernameExists
		}
		return nil, err
	}

	tokens, err := s.GenerateTokenPair(u, 0, "")
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "user registered", "user_id", u.ID, "email", req.Email)

	return &LoginResult{
		User:         u,
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
	}, nil
}
