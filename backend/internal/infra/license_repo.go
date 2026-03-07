package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"gorm.io/gorm"
)

// Compile-time interface check.
var _ billing.LicenseRepository = (*licenseRepo)(nil)

type licenseRepo struct {
	db *gorm.DB
}

// NewLicenseRepository creates a new GORM-based license repository.
func NewLicenseRepository(db *gorm.DB) billing.LicenseRepository {
	return &licenseRepo{db: db}
}

func (r *licenseRepo) GetByKey(ctx context.Context, licenseKey string) (*billing.License, error) {
	var license billing.License
	err := r.db.WithContext(ctx).Where("license_key = ?", licenseKey).First(&license).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &license, nil
}

func (r *licenseRepo) GetActiveLicense(ctx context.Context) (*billing.License, error) {
	var license billing.License
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("created_at DESC").
		First(&license).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &license, nil
}

func (r *licenseRepo) Save(ctx context.Context, license *billing.License) error {
	return r.db.WithContext(ctx).Save(license).Error
}

func (r *licenseRepo) Create(ctx context.Context, license *billing.License) error {
	return r.db.WithContext(ctx).Create(license).Error
}

func (r *licenseRepo) DeactivateAll(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Model(&billing.License{}).
		Where("is_active = ?", true).
		Update("is_active", false).Error
}
