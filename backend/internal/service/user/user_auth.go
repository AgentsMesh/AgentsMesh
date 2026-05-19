package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) Authenticate(ctx context.Context, email, password string) (*user.User, error) {
	u, err := s.GetByEmail(ctx, email)
	if err != nil {
		slog.WarnContext(ctx, "authentication failed: user not found", "email", email)
		return nil, ErrInvalidCredentials
	}

	if !u.IsActive {
		slog.WarnContext(ctx, "authentication failed: user inactive", "user_id", u.ID, "email", email)
		return nil, ErrUserInactive
	}

	if u.PasswordHash == nil || *u.PasswordHash == "" {
		slog.WarnContext(ctx, "authentication failed: no password set", "user_id", u.ID)
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		slog.WarnContext(ctx, "authentication failed: wrong password", "user_id", u.ID)
		return nil, ErrInvalidCredentials
	}

	s.RecordLogin(ctx, u.ID)

	slog.InfoContext(ctx, "user authenticated", "user_id", u.ID, "email", email)
	return u, nil
}

func (s *Service) RecordLogin(ctx context.Context, userID int64) {
	now := time.Now()
	if err := s.repo.UpdateUserField(ctx, userID, "last_login_at", now); err != nil {
		slog.WarnContext(ctx, "failed to update last_login_at", "user_id", userID, "error", err)
	}
}

func (s *Service) SetEmailVerificationToken(ctx context.Context, userID int64) (string, error) {
	token, err := generateToken()
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate email verification token", "user_id", userID, "error", err)
		return "", err
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	err = s.repo.UpdateUser(ctx, userID, map[string]interface{}{
		"email_verification_token":      token,
		"email_verification_expires_at": expiresAt,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to save email verification token", "user_id", userID, "error", err)
	}

	return token, err
}

func (s *Service) VerifyEmail(ctx context.Context, token string) (*user.User, error) {
	u, err := s.repo.GetByVerificationToken(ctx, token)
	if err != nil {
		slog.WarnContext(ctx, "email verification failed: invalid token")
		return nil, ErrInvalidVerificationToken
	}

	if u.EmailVerificationExpiresAt == nil || time.Now().After(*u.EmailVerificationExpiresAt) {
		slog.WarnContext(ctx, "email verification failed: token expired", "user_id", u.ID)
		return nil, ErrInvalidVerificationToken
	}

	if u.IsEmailVerified {
		return nil, ErrEmailAlreadyVerified
	}

	err = s.repo.UpdateUser(ctx, u.ID, map[string]interface{}{
		"is_email_verified":             true,
		"email_verification_token":      nil,
		"email_verification_expires_at": nil,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to mark email as verified", "user_id", u.ID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "email verified", "user_id", u.ID)
	u.IsEmailVerified = true
	return u, nil
}

func (s *Service) SetPasswordResetToken(ctx context.Context, email string) (string, *user.User, error) {
	u, err := s.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, ErrUserNotFound
	}

	token, err := generateToken()
	if err != nil {
		return "", nil, err
	}

	expiresAt := time.Now().Add(1 * time.Hour)

	err = s.repo.UpdateUser(ctx, u.ID, map[string]interface{}{
		"password_reset_token":      token,
		"password_reset_expires_at": expiresAt,
	})

	return token, u, err
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) (*user.User, error) {
	u, err := s.repo.GetByResetToken(ctx, token)
	if err != nil {
		slog.WarnContext(ctx, "password reset failed: invalid token")
		return nil, ErrInvalidResetToken
	}

	if u.PasswordResetExpiresAt == nil || time.Now().After(*u.PasswordResetExpiresAt) {
		slog.WarnContext(ctx, "password reset failed: token expired", "user_id", u.ID)
		return nil, ErrInvalidResetToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(ctx, "failed to hash new password", "user_id", u.ID, "error", err)
		return nil, err
	}

	err = s.repo.UpdateUser(ctx, u.ID, map[string]interface{}{
		"password_hash":             string(hash),
		"password_reset_token":      nil,
		"password_reset_expires_at": nil,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to update password", "user_id", u.ID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "password reset completed", "user_id", u.ID)
	return u, nil
}

func (s *Service) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	u, err := s.repo.GetByVerificationToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidVerificationToken
	}
	return u, nil
}
