package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"golang.org/x/crypto/bcrypt"
)

// CreateRegistrationToken creates a new registration token
func (s *Service) CreateRegistrationToken(ctx context.Context, orgID, userID int64, description string, maxUses *int, expiresAt *time.Time) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)

	// Hash the token for storage
	tokenHash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	regToken := &runner.RegistrationToken{
		OrganizationID: orgID,
		TokenHash:      string(tokenHash),
		Description:    description,
		CreatedByID:    userID,
		IsActive:       true,
		MaxUses:        maxUses,
		UsedCount:      0,
		ExpiresAt:      expiresAt,
	}

	if err := s.db.WithContext(ctx).Create(regToken).Error; err != nil {
		return "", err
	}

	return token, nil
}

// ValidateRegistrationToken validates a registration token
func (s *Service) ValidateRegistrationToken(ctx context.Context, token string) (*runner.RegistrationToken, error) {
	var tokens []runner.RegistrationToken
	if err := s.db.WithContext(ctx).Where("is_active = ?", true).Find(&tokens).Error; err != nil {
		return nil, err
	}

	for _, t := range tokens {
		if err := bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(token)); err == nil {
			// Check expiration
			if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
				return nil, ErrTokenExpired
			}

			// Check usage
			if t.MaxUses != nil && t.UsedCount >= *t.MaxUses {
				return nil, ErrTokenExhausted
			}

			return &t, nil
		}
	}

	return nil, ErrInvalidToken
}

// ListRegistrationTokens lists registration tokens for an organization
func (s *Service) ListRegistrationTokens(ctx context.Context, orgID int64) ([]*runner.RegistrationToken, error) {
	var tokens []*runner.RegistrationToken
	if err := s.db.WithContext(ctx).Where("organization_id = ?", orgID).Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

// RevokeRegistrationToken revokes a registration token
func (s *Service) RevokeRegistrationToken(ctx context.Context, tokenID int64) error {
	return s.db.WithContext(ctx).Model(&runner.RegistrationToken{}).
		Where("id = ?", tokenID).
		Update("is_active", false).Error
}
