package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

// Authenticate authenticates a user by email and password
func (s *Service) Authenticate(ctx context.Context, email, password string) (*user.User, error) {
	u, err := s.GetByEmail(ctx, email)
	if err != nil {
		slog.Warn("authentication failed: user not found", "email", email)
		return nil, ErrInvalidCredentials
	}

	if !u.IsActive {
		slog.Warn("authentication failed: user inactive", "user_id", u.ID, "email", email)
		return nil, ErrUserInactive
	}

	if u.PasswordHash == nil || *u.PasswordHash == "" {
		slog.Warn("authentication failed: no password set", "user_id", u.ID)
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		slog.Warn("authentication failed: wrong password", "user_id", u.ID)
		return nil, ErrInvalidCredentials
	}

	// Update last login
	now := time.Now()
	_ = s.repo.UpdateUserField(ctx, u.ID, "last_login_at", now)

	slog.Info("user authenticated", "user_id", u.ID, "email", email)
	return u, nil
}

// SetEmailVerificationToken generates and sets a verification token for the user
// Returns the token to be sent via email
func (s *Service) SetEmailVerificationToken(ctx context.Context, userID int64) (string, error) {
	token, err := generateToken()
	if err != nil {
		slog.Error("failed to generate email verification token", "user_id", userID, "error", err)
		return "", err
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	err = s.repo.UpdateUser(ctx, userID, map[string]interface{}{
		"email_verification_token":      token,
		"email_verification_expires_at": expiresAt,
	})
	if err != nil {
		slog.Error("failed to save email verification token", "user_id", userID, "error", err)
	}

	return token, err
}

// VerifyEmail verifies a user's email using the verification token
func (s *Service) VerifyEmail(ctx context.Context, token string) (*user.User, error) {
	u, err := s.repo.GetByVerificationToken(ctx, token)
	if err != nil {
		slog.Warn("email verification failed: invalid token")
		return nil, ErrInvalidVerificationToken
	}

	// Check if token has expired
	if u.EmailVerificationExpiresAt == nil || time.Now().After(*u.EmailVerificationExpiresAt) {
		slog.Warn("email verification failed: token expired", "user_id", u.ID)
		return nil, ErrInvalidVerificationToken
	}

	// Check if already verified
	if u.IsEmailVerified {
		return nil, ErrEmailAlreadyVerified
	}

	// Mark as verified and clear token
	err = s.repo.UpdateUser(ctx, u.ID, map[string]interface{}{
		"is_email_verified":             true,
		"email_verification_token":      nil,
		"email_verification_expires_at": nil,
	})
	if err != nil {
		slog.Error("failed to mark email as verified", "user_id", u.ID, "error", err)
		return nil, err
	}

	slog.Info("email verified", "user_id", u.ID)
	u.IsEmailVerified = true
	return u, nil
}

// SetPasswordResetToken generates and sets a password reset token for the user
// Returns the token to be sent via email
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

// ResetPassword resets the user's password using the reset token
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) (*user.User, error) {
	u, err := s.repo.GetByResetToken(ctx, token)
	if err != nil {
		slog.Warn("password reset failed: invalid token")
		return nil, ErrInvalidResetToken
	}

	// Check if token has expired
	if u.PasswordResetExpiresAt == nil || time.Now().After(*u.PasswordResetExpiresAt) {
		slog.Warn("password reset failed: token expired", "user_id", u.ID)
		return nil, ErrInvalidResetToken
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash new password", "user_id", u.ID, "error", err)
		return nil, err
	}

	// Update password and clear reset token
	err = s.repo.UpdateUser(ctx, u.ID, map[string]interface{}{
		"password_hash":             string(hash),
		"password_reset_token":      nil,
		"password_reset_expires_at": nil,
	})
	if err != nil {
		slog.Error("failed to update password", "user_id", u.ID, "error", err)
		return nil, err
	}

	slog.Info("password reset completed", "user_id", u.ID)
	return u, nil
}

// GetByVerificationToken returns a user by their verification token
func (s *Service) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	u, err := s.repo.GetByVerificationToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidVerificationToken
	}
	return u, nil
}
