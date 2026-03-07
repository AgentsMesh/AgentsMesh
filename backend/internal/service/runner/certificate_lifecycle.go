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
	r, err := s.repo.GetByNodeID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrRunnerNotFound
	}

	// Verify certificate serial matches
	if r.CertSerialNumber == nil || *r.CertSerialNumber != oldSerial {
		return nil, ErrCertificateMismatch
	}

	// Get org slug
	orgSlug, err := s.repo.GetOrgSlug(ctx, r.OrganizationID)
	if err != nil {
		return nil, err
	}
	if orgSlug == "" {
		return nil, fmt.Errorf("organization not found")
	}

	// Issue new certificate
	certInfo, err := pkiService.IssueRunnerCertificate(nodeID, orgSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Revoke old certificate (best-effort)
	if err := s.repo.RevokeCertificate(ctx, oldSerial, "renewed"); err != nil {
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
	if err := s.repo.CreateCertificate(ctx, cert); err != nil {
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	// Update runner (CAS: only succeeds if cert_serial_number still matches oldSerial).
	rowsAffected, err := s.repo.UpdateFieldsCAS(ctx, r.ID, "cert_serial_number", oldSerial, map[string]interface{}{
		"cert_serial_number": certInfo.SerialNumber,
		"cert_expires_at":    certInfo.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update runner: %w", err)
	}
	if rowsAffected == 0 {
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
	return s.repo.RevokeCertificate(ctx, serialNumber, reason)
}

// IsCertificateRevoked checks if a certificate is revoked.
func (s *Service) IsCertificateRevoked(ctx context.Context, serialNumber string) (bool, error) {
	cert, err := s.repo.GetCertificateBySerial(ctx, serialNumber)
	if err != nil {
		return false, err
	}
	if cert == nil {
		// Certificate not found in DB - not revoked (might be legacy)
		return false, nil
	}
	return cert.IsRevoked(), nil
}
