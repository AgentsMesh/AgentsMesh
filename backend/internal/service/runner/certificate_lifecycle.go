package runner

import (
	"context"
	"fmt"
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
		return nil, fmt.Errorf("runner not found")
	}

	// Verify certificate serial matches
	if r.CertSerialNumber == nil || *r.CertSerialNumber != oldSerial {
		return nil, fmt.Errorf("certificate mismatch")
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

	// Revoke old certificate
	now := time.Now()
	reason := "renewed"
	if err := s.db.WithContext(ctx).Model(&runner.Certificate{}).
		Where("serial_number = ?", oldSerial).
		Updates(map[string]interface{}{
			"revoked_at":        now,
			"revocation_reason": reason,
		}).Error; err != nil {
		// Log but don't fail - old cert may not exist in DB
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

	// Update runner
	if err := s.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ?", r.ID).
		Updates(map[string]interface{}{
			"cert_serial_number": certInfo.SerialNumber,
			"cert_expires_at":    certInfo.ExpiresAt,
		}).Error; err != nil {
		return nil, fmt.Errorf("failed to update runner: %w", err)
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
