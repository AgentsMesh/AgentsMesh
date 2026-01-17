package runner

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"gorm.io/gorm"
)

// RegisterRunner registers a new runner.
// Note: This creates the runner record only. For secure communication,
// use gRPC/mTLS registration to obtain certificates.
func (s *Service) RegisterRunner(ctx context.Context, token, nodeID, description string, maxPods int) (*runner.Runner, error) {
	// Validate token
	regToken, err := s.ValidateRegistrationToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check runner quota before registration
	if s.billingService != nil {
		if err := s.billingService.CheckQuota(ctx, regToken.OrganizationID, "runners", 1); err != nil {
			if err == billing.ErrQuotaExceeded {
				return nil, ErrRunnerQuotaExceeded
			}
			return nil, err
		}
	}

	// Check if runner already exists
	var existing runner.Runner
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND node_id = ?", regToken.OrganizationID, nodeID).First(&existing).Error; err == nil {
		return nil, ErrRunnerAlreadyExists
	}

	// Create runner (no auth_token - using mTLS certificates instead)
	r := &runner.Runner{
		OrganizationID:    regToken.OrganizationID,
		NodeID:            nodeID,
		Description:       description,
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
		return nil, err
	}

	return r, nil
}

// DeleteRunner deletes a runner
func (s *Service) DeleteRunner(ctx context.Context, runnerID int64) error {
	return s.db.WithContext(ctx).Delete(&runner.Runner{}, runnerID).Error
}
