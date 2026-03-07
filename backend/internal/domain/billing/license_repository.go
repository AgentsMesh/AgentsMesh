package billing

import "context"

// LicenseRepository defines data access for License entities.
type LicenseRepository interface {
	// GetByKey returns a license by its key. Returns (nil, nil) when not found.
	GetByKey(ctx context.Context, licenseKey string) (*License, error)

	// GetActiveLicense returns the most recent active license.
	// Returns (nil, nil) when not found.
	GetActiveLicense(ctx context.Context) (*License, error)

	// Save upserts a license (insert or update all fields).
	Save(ctx context.Context, license *License) error

	// Create inserts a new license record.
	Create(ctx context.Context, license *License) error

	// DeactivateAll sets is_active=false on all active licenses.
	DeactivateAll(ctx context.Context) error
}
