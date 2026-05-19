package billing

import "context"

type LicenseRepository interface {
	GetByKey(ctx context.Context, licenseKey string) (*License, error)

	GetActiveLicense(ctx context.Context) (*License, error)

	Save(ctx context.Context, license *License) error

	Create(ctx context.Context, license *License) error

	DeactivateAll(ctx context.Context) error
}
