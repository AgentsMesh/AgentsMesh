package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
)

// ==================== Certificate Renewal ====================

// RenewCertificateResponse represents the certificate renewal response.
type RenewCertificateResponse struct {
	Certificate string    `json:"certificate"`
	PrivateKey  string    `json:"private_key"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// RenewCertificate renews a runner's certificate.
// Called when certificate is about to expire (within 30 days).
func (s *Service) RenewCertificate(ctx context.Context, nodeID, oldSerial string, pkiService interfaces.PKICertificateIssuer) (*RenewCertificateResponse, error) {
	// Find runner by node_id
	var r runner.Runner
	if err := s.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&r).Error; err != nil {
		return nil, ErrRunnerNotFound
	}

	// Verify certificate serial matches
	if r.CertSerialNumber == nil || *r.CertSerialNumber != oldSerial {
		return nil, ErrCertificateMismatch
	}

	// Get org slug
	var org struct {
		Slug string
	}
	if err := s.db.WithContext(ctx).Table("organizations").
		Select("slug").
		Where("id = ?", r.OrganizationID).
		First(&org).Error; err != nil {
		return nil, fmt.Errorf("organization not found")
	}

	// Issue new certificate
	certInfo, err := pkiService.IssueRunnerCertificate(nodeID, org.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Revoke old certificate (best-effort: old cert may not exist in DB for legacy runners)
	now := time.Now()
	reason := "renewed"
	if err := s.db.WithContext(ctx).Model(&runner.Certificate{}).
		Where("serial_number = ?", oldSerial).
		Updates(map[string]interface{}{
			"revoked_at":        now,
			"revocation_reason": reason,
		}).Error; err != nil {
		slog.Warn("Failed to revoke old certificate during renewal",
			"node_id", nodeID, "old_serial", oldSerial, "error", err)
	}

	// Save new certificate
	cert := &runner.Certificate{
		RunnerID:     r.ID,
		SerialNumber: certInfo.SerialNumber,
		Fingerprint:  certInfo.Fingerprint,
		IssuedAt:     certInfo.IssuedAt,
		ExpiresAt:    certInfo.ExpiresAt,
	}
	if err := s.db.WithContext(ctx).Create(cert).Error; err != nil {
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	// Update runner (CAS: only succeeds if cert_serial_number still matches oldSerial).
	// This serializes concurrent renewals — at most one wins. The loser's issued cert
	// remains in the certificates table but is never referenced by the runner (harmless orphan).
	updateResult := s.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ? AND cert_serial_number = ?", r.ID, oldSerial).
		Updates(map[string]interface{}{
			"cert_serial_number": certInfo.SerialNumber,
			"cert_expires_at":    certInfo.ExpiresAt,
		})
	if updateResult.Error != nil {
		return nil, fmt.Errorf("failed to update runner: %w", updateResult.Error)
	}
	if updateResult.RowsAffected == 0 {
		// Another concurrent renewal already completed; discard this duplicate.
		slog.Warn("Concurrent certificate renewal detected, discarding duplicate",
			"node_id", nodeID, "orphaned_serial", certInfo.SerialNumber)
		return nil, ErrCertificateMismatch
	}

	return &RenewCertificateResponse{
		Certificate: string(certInfo.CertPEM),
		PrivateKey:  string(certInfo.KeyPEM),
		ExpiresAt:   certInfo.ExpiresAt,
	}, nil
}

// ==================== Certificate Revocation ====================

// RevokeCertificate revokes a runner's certificate.
func (s *Service) RevokeCertificate(ctx context.Context, serialNumber, reason string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&runner.Certificate{}).
		Where("serial_number = ?", serialNumber).
		Updates(map[string]interface{}{
			"revoked_at":        now,
			"revocation_reason": reason,
		}).Error
}

// IsCertificateRevoked checks if a certificate is revoked.
func (s *Service) IsCertificateRevoked(ctx context.Context, serialNumber string) (bool, error) {
	var cert runner.Certificate
	if err := s.db.WithContext(ctx).Where("serial_number = ?", serialNumber).First(&cert).Error; err != nil {
		// Certificate not found in DB - not revoked (might be legacy)
		return false, nil
	}
	return cert.IsRevoked(), nil
}
