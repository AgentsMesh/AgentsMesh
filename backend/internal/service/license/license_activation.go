package license

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

func (s *Service) ActivateLicense(ctx context.Context, licenseData []byte) error {
	license, err := s.ParseAndVerify(licenseData)
	if err != nil {
		return fmt.Errorf("invalid license: %w", err)
	}

	activatedAt := time.Now()
	expiresAt := license.ExpiresAt

	features := make(billing.Features)
	for _, f := range license.Features {
		features[f] = true
	}

	dbLicense := &billing.License{
		LicenseKey:       license.LicenseKey,
		OrganizationName: license.OrganizationName,
		ContactEmail:     license.ContactEmail,
		PlanName:         license.Plan,
		MaxUsers:         license.Limits.MaxUsers,
		MaxRunners:       license.Limits.MaxRunners,
		MaxRepositories:  license.Limits.MaxRepositories,
		MaxConcurrentPods: license.Limits.MaxPodMinutes,
		Features:          features,
		IssuedAt:          license.IssuedAt,
		ExpiresAt:         &expiresAt,
		Signature:         license.Signature,
		ActivatedAt:       &activatedAt,
		IsActive:          true,
	}

	if err := s.repo.DeactivateAll(ctx); err != nil {
		s.logger.Warn("failed to deactivate existing licenses", "error", err)
	}

	if err := s.repo.Create(ctx, dbLicense); err != nil {
		return fmt.Errorf("failed to save license: %w", err)
	}

	s.mu.Lock()
	s.currentLicense = license
	s.lastCheck = time.Now()
	s.mu.Unlock()

	s.logger.Info("license activated",
		"license_key", license.LicenseKey,
		"organization", license.OrganizationName,
		"plan", license.Plan,
		"expires_at", license.ExpiresAt,
	)

	return nil
}
