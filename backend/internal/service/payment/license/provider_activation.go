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

func (p *Provider) GetLicenseStatus(ctx context.Context) (*types.LicenseStatus, error) {
	license, err := p.repo.GetActiveLicense(ctx)
	if err != nil {
		return nil, err
	}
	if license != nil {
		return p.licenseToStatus(license), nil
	}

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

func (p *Provider) ActivateLicense(ctx context.Context, licenseKey string, orgID int64) error {
	license, err := p.repo.GetByKey(ctx, licenseKey)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get license for activation", "license_key", licenseKey, "org_id", orgID, "error", err)
		return err
	}
	if license == nil {
		return ErrLicenseNotFound
	}

	if !license.IsValid() {
		if license.RevokedAt != nil {
			slog.WarnContext(ctx, "attempted to activate revoked license", "license_key", licenseKey, "org_id", orgID)
			return ErrLicenseRevoked
		}
		if license.ExpiresAt != nil && time.Now().After(*license.ExpiresAt) {
			slog.WarnContext(ctx, "attempted to activate expired license", "license_key", licenseKey, "org_id", orgID)
			return ErrLicenseExpired
		}
		return ErrInvalidLicense
	}

	if license.IsActivated() && *license.ActivatedOrgID != orgID {
		slog.WarnContext(ctx, "license already activated for another org", "license_key", licenseKey, "org_id", orgID, "activated_org_id", *license.ActivatedOrgID)
		return ErrAlreadyActivated
	}

	now := time.Now()
	license.ActivatedAt = &now
	license.ActivatedOrgID = &orgID
	license.LastVerifiedAt = &now

	if err := p.repo.Save(ctx, license); err != nil {
		slog.ErrorContext(ctx, "failed to save activated license", "license_key", licenseKey, "org_id", orgID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "license activated", "license_key", licenseKey, "org_id", orgID)
	return nil
}

func (p *Provider) ActivateLicenseFromFile(ctx context.Context, licenseData []byte, orgID int64) (*billing.License, error) {
	license, err := p.VerifyLicense(ctx, licenseData)
	if err != nil {
		return nil, err
	}

	existing, err := p.repo.GetByKey(ctx, license.LicenseKey)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check existing license", "license_key", license.LicenseKey, "org_id", orgID, "error", err)
		return nil, err
	}

	if existing != nil {
		if existing.IsActivated() && *existing.ActivatedOrgID != orgID {
			slog.WarnContext(ctx, "license file already activated for another org", "license_key", license.LicenseKey, "org_id", orgID)
			return nil, ErrAlreadyActivated
		}
		now := time.Now()
		existing.ActivatedAt = &now
		existing.ActivatedOrgID = &orgID
		existing.LastVerifiedAt = &now
		if err := p.repo.Save(ctx, existing); err != nil {
			slog.ErrorContext(ctx, "failed to update existing license from file", "license_key", license.LicenseKey, "org_id", orgID, "error", err)
			return nil, err
		}
		slog.InfoContext(ctx, "existing license activated from file", "license_key", license.LicenseKey, "org_id", orgID)
		return existing, nil
	}

	now := time.Now()
	license.ActivatedAt = &now
	license.ActivatedOrgID = &orgID
	license.LastVerifiedAt = &now

	if err := p.repo.Create(ctx, license); err != nil {
		slog.ErrorContext(ctx, "failed to create license from file", "license_key", license.LicenseKey, "org_id", orgID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "new license activated from file", "license_key", license.LicenseKey, "org_id", orgID)
	return license, nil
}
