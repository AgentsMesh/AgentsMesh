package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RegisterRunner registers a new runner
func (s *Service) RegisterRunner(ctx context.Context, token, nodeID, description string, maxPods int) (*runner.Runner, string, error) {
	// Validate token
	regToken, err := s.ValidateRegistrationToken(ctx, token)
	if err != nil {
		return nil, "", err
	}

	// Check runner quota before registration
	if s.billingService != nil {
		if err := s.billingService.CheckQuota(ctx, regToken.OrganizationID, "runners", 1); err != nil {
			if err == billing.ErrQuotaExceeded {
				return nil, "", ErrRunnerQuotaExceeded
			}
			return nil, "", err
		}
	}

	// Check if runner already exists
	var existing runner.Runner
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND node_id = ?", regToken.OrganizationID, nodeID).First(&existing).Error; err == nil {
		return nil, "", ErrRunnerAlreadyExists
	}

	// Generate auth token
	authTokenBytes := make([]byte, 32)
	if _, err := rand.Read(authTokenBytes); err != nil {
		return nil, "", err
	}
	authToken := hex.EncodeToString(authTokenBytes)

	authTokenHash, err := bcrypt.GenerateFromPassword([]byte(authToken), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	// Create runner
	r := &runner.Runner{
		OrganizationID:    regToken.OrganizationID,
		NodeID:            nodeID,
		Description:       description,
		AuthTokenHash:     string(authTokenHash),
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: maxPods,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(r).Error; err != nil {
			return err
		}

		// Increment token usage
		return tx.Model(regToken).Update("used_count", gorm.Expr("used_count + 1")).Error
	})

	if err != nil {
		return nil, "", err
	}

	return r, authToken, nil
}

// DeleteRunner deletes a runner
func (s *Service) DeleteRunner(ctx context.Context, runnerID int64) error {
	return s.db.WithContext(ctx).Delete(&runner.Runner{}, runnerID).Error
}
