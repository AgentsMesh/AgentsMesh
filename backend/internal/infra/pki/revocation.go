package pki

import (
	"context"
	"time"
)

// RevocationChecker checks if a certificate has been revoked.
// It queries the database at connection time only (not on every request).
type RevocationChecker struct {
	repo RevocationRepository
}

// RevocationRepository defines the interface for certificate revocation data access.
type RevocationRepository interface {
	// IsRevoked checks if a certificate with the given serial number is revoked.
	IsRevoked(ctx context.Context, serialNumber string) (bool, error)

	// GetRevokedSerials returns all revoked serial numbers (for caching if needed).
	GetRevokedSerials(ctx context.Context) ([]string, error)

	// Revoke marks a certificate as revoked.
	Revoke(ctx context.Context, serialNumber string, reason string) error
}

// RevokedCertificate represents a revoked certificate record.
type RevokedCertificate struct {
	ID               int64
	RunnerID         int64
	SerialNumber     string
	Fingerprint      string
	IssuedAt         time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	RevocationReason string
	CreatedAt        time.Time
}

// NewRevocationChecker creates a new revocation checker.
func NewRevocationChecker(repo RevocationRepository) *RevocationChecker {
	return &RevocationChecker{
		repo: repo,
	}
}

// IsRevoked checks if a certificate is revoked.
// This is called only at connection establishment, not on every message.
func (c *RevocationChecker) IsRevoked(ctx context.Context, serialNumber string) (bool, error) {
	if c.repo == nil {
		// If no repository is configured, assume not revoked (development mode)
		return false, nil
	}
	return c.repo.IsRevoked(ctx, serialNumber)
}

// Revoke marks a certificate as revoked.
func (c *RevocationChecker) Revoke(ctx context.Context, serialNumber string, reason string) error {
	if c.repo == nil {
		return nil
	}
	return c.repo.Revoke(ctx, serialNumber, reason)
}
