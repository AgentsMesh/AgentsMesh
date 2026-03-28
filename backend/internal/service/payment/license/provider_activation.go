package license

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
)

// GetLicenseStatus returns the current license status
func (p *Provider) GetLicenseStatus(ctx context.Context) (*types.LicenseStatus, error) {
	// Try to load from database first (if license was activated)
	license, err := p.repo.GetActiveLicense(ctx)
	if err != nil {
		return nil, err
	}
	if license != nil {
		return p.licenseToStatus(license), nil
	}

	// Try to load from file if configured
	if p.config.LicenseFilePath != "" {
		licenseData, err := os.ReadFile(p.config.LicenseFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				return &types.LicenseStatus{
					IsValid: false,
					Message: "No license found",
				}, nil
			}
			return nil, fmt.Errorf("failed to read license file: %w", err)
		}

		verifiedLicense, err := p.VerifyLicense(ctx, licenseData)
		if err != nil {
			return &types.LicenseStatus{
				IsValid: false,
				Message: err.Error(),
			}, nil
		}

		return p.licenseToStatus(verifiedLicense), nil
	}

	return &types.LicenseStatus{
		IsValid: false,
		Message: "No license configured",
	}, nil
}

// ActivateLicense activates a license for an organization
func (p *Provider) ActivateLicense(ctx context.Context, licenseKey string, orgID int64) error {
	// Find the license by key
	license, err := p.repo.GetByKey(ctx, licenseKey)
	if err != nil {
		slog.Error("failed to get license for activation", "license_key", licenseKey, "org_id", orgID, "error", err)
		return err
	}
	if license == nil {
		return ErrLicenseNotFound
	}

	// Check if license is valid
	if !license.IsValid() {
		if license.RevokedAt != nil {
			slog.Warn("attempted to activate revoked license", "license_key", licenseKey, "org_id", orgID)
			return ErrLicenseRevoked
		}
		if license.ExpiresAt != nil && time.Now().After(*license.ExpiresAt) {
			slog.Warn("attempted to activate expired license", "license_key", licenseKey, "org_id", orgID)
			return ErrLicenseExpired
		}
		return ErrInvalidLicense
	}

	// Check if already activated for another org
	if license.IsActivated() && *license.ActivatedOrgID != orgID {
		slog.Warn("license already activated for another org", "license_key", licenseKey, "org_id", orgID, "activated_org_id", *license.ActivatedOrgID)
		return ErrAlreadyActivated
	}

	// Activate the license
	now := time.Now()
	license.ActivatedAt = &now
	license.ActivatedOrgID = &orgID
	license.LastVerifiedAt = &now

	if err := p.repo.Save(ctx, license); err != nil {
		slog.Error("failed to save activated license", "license_key", licenseKey, "org_id", orgID, "error", err)
		return err
	}
	slog.Info("license activated", "license_key", licenseKey, "org_id", orgID)
	return nil
}

// ActivateLicenseFromFile activates a license from file data
func (p *Provider) ActivateLicenseFromFile(ctx context.Context, licenseData []byte, orgID int64) (*billing.License, error) {
	// Verify the license first
	license, err := p.VerifyLicense(ctx, licenseData)
	if err != nil {
		return nil, err
	}

	// Check if this license key already exists
	existing, err := p.repo.GetByKey(ctx, license.LicenseKey)
	if err != nil {
		slog.Error("failed to check existing license", "license_key", license.LicenseKey, "org_id", orgID, "error", err)
		return nil, err
	}

	if existing != nil {
		// License exists - check if it can be activated
		if existing.IsActivated() && *existing.ActivatedOrgID != orgID {
			slog.Warn("license file already activated for another org", "license_key", license.LicenseKey, "org_id", orgID)
			return nil, ErrAlreadyActivated
		}
		// Update existing license
		now := time.Now()
		existing.ActivatedAt = &now
		existing.ActivatedOrgID = &orgID
		existing.LastVerifiedAt = &now
		if err := p.repo.Save(ctx, existing); err != nil {
			slog.Error("failed to update existing license from file", "license_key", license.LicenseKey, "org_id", orgID, "error", err)
			return nil, err
		}
		slog.Info("existing license activated from file", "license_key", license.LicenseKey, "org_id", orgID)
		return existing, nil
	}

	// Create new license record
	now := time.Now()
	license.ActivatedAt = &now
	license.ActivatedOrgID = &orgID
	license.LastVerifiedAt = &now

	if err := p.repo.Create(ctx, license); err != nil {
		slog.Error("failed to create license from file", "license_key", license.LicenseKey, "org_id", orgID, "error", err)
		return nil, err
	}

	slog.Info("new license activated from file", "license_key", license.LicenseKey, "org_id", orgID)
	return license, nil
}
