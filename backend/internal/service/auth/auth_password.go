package auth

import (
	"context"

	userService "github.com/AgentsMesh/AgentsMesh/backend/internal/service/user"
)

// Login authenticates user and returns tokens
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	u, err := s.userService.Authenticate(ctx, email, password)
	if err != nil {
		if err == userService.ErrInvalidCredentials {
			return nil, ErrInvalidCredentials
		}
		if err == userService.ErrUserInactive {
			return nil, ErrUserDisabled
		}
		return nil, err
	}

	tokens, err := s.GenerateTokenPair(u, 0, "")
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         u,
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
	}, nil
}

// Register creates a new user and returns tokens
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*LoginResult, error) {
	u, err := s.userService.Create(ctx, &userService.CreateRequest{
		Email:    req.Email,
		Username: req.Username,
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		if err == userService.ErrEmailAlreadyExists {
			return nil, ErrEmailExists
		}
		if err == userService.ErrUsernameExists {
			return nil, ErrUsernameExists
		}
		return nil, err
	}

	tokens, err := s.GenerateTokenPair(u, 0, "")
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         u,
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
	}, nil
}
