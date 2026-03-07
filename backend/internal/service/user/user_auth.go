package user

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

// Authenticate authenticates a user by email and password
func (s *Service) Authenticate(ctx context.Context, email, password string) (*user.User, error) {
	u, err := s.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !u.IsActive {
		return nil, ErrUserInactive
	}

	if u.PasswordHash == nil || *u.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	now := time.Now()
	_ = s.repo.UpdateUserField(ctx, u.ID, "last_login_at", now)

	return u, nil
}

// SetEmailVerificationToken generates and sets a verification token for the user
// Returns the token to be sent via email
func (s *Service) SetEmailVerificationToken(ctx context.Context, userID int64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	err = s.repo.UpdateUser(ctx, userID, map[string]interface{}{
		"email_verification_token":      token,
		"email_verification_expires_at": expiresAt,
	})

	return token, err
}

// VerifyEmail verifies a user's email using the verification token
func (s *Service) VerifyEmail(ctx context.Context, token string) (*user.User, error) {
	u, err := s.repo.GetByVerificationToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidVerificationToken
	}

	// Check if token has expired
	if u.EmailVerificationExpiresAt == nil || time.Now().After(*u.EmailVerificationExpiresAt) {
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
		return nil, err
	}

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
		return nil, ErrInvalidResetToken
	}

	// Check if token has expired
	if u.PasswordResetExpiresAt == nil || time.Now().After(*u.PasswordResetExpiresAt) {
		return nil, ErrInvalidResetToken
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Update password and clear reset token
	err = s.repo.UpdateUser(ctx, u.ID, map[string]interface{}{
		"password_hash":             string(hash),
		"password_reset_token":      nil,
		"password_reset_expires_at": nil,
	})
	if err != nil {
		return nil, err
	}

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
