package pki

import (
	"context"
	"time"
)

type RevocationChecker struct {
	repo RevocationRepository
}

type RevocationRepository interface {
	IsRevoked(ctx context.Context, serialNumber string) (bool, error)

	GetRevokedSerials(ctx context.Context) ([]string, error)

	Revoke(ctx context.Context, serialNumber string, reason string) error
}

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

func NewRevocationChecker(repo RevocationRepository) *RevocationChecker {
	return &RevocationChecker{
		repo: repo,
	}
}

func (c *RevocationChecker) IsRevoked(ctx context.Context, serialNumber string) (bool, error) {
	if c.repo == nil {
		return false, nil
	}
	return c.repo.IsRevoked(ctx, serialNumber)
}

func (c *RevocationChecker) Revoke(ctx context.Context, serialNumber string, reason string) error {
	if c.repo == nil {
		return nil
	}
	return c.repo.Revoke(ctx, serialNumber, reason)
}
