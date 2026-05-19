package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
)

type RenewCertificateResponse struct {
	Certificate string    `json:"certificate"`
	PrivateKey  string    `json:"private_key"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// RenewCertificate is the cert-renewal path — must run when expiry is within 30d.
func (s *Service) RenewCertificate(ctx context.Context, nodeID, oldSerial string, pkiService interfaces.PKICertificateIssuer) (*RenewCertificateResponse, error) {
	r, err := s.repo.GetByNodeID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrRunnerNotFound
	}

	if r.CertSerialNumber == nil || *r.CertSerialNumber != oldSerial {
		return nil, ErrCertificateMismatch
	}

	orgSlug, err := s.repo.GetOrgSlug(ctx, r.OrganizationID)
	if err != nil {
		return nil, err
	}
	if orgSlug == "" {
		return nil, fmt.Errorf("organization not found")
	}

	certInfo, err := pkiService.IssueRunnerCertificate(nodeID, orgSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	if err := s.repo.RevokeCertificate(ctx, oldSerial, "renewed"); err != nil {
		slog.WarnContext(ctx, "Failed to revoke old certificate during renewal",
			"node_id", nodeID, "old_serial", oldSerial, "error", err)
	}

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

	rowsAffected, err := s.repo.UpdateFieldsCAS(ctx, r.ID, "cert_serial_number", oldSerial, map[string]interface{}{
		"cert_serial_number": certInfo.SerialNumber,
		"cert_expires_at":    certInfo.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update runner: %w", err)
	}
	if rowsAffected == 0 {
		slog.WarnContext(ctx, "Concurrent certificate renewal detected, discarding duplicate",
			"node_id", nodeID, "orphaned_serial", certInfo.SerialNumber)
		return nil, ErrCertificateMismatch
	}

	return &RenewCertificateResponse{
		Certificate: string(certInfo.CertPEM),
		PrivateKey:  string(certInfo.KeyPEM),
		ExpiresAt:   certInfo.ExpiresAt,
	}, nil
}

func (s *Service) RevokeCertificate(ctx context.Context, serialNumber, reason string) error {
	return s.repo.RevokeCertificate(ctx, serialNumber, reason)
}

func (s *Service) IsCertificateRevoked(ctx context.Context, serialNumber string) (bool, error) {
	cert, err := s.repo.GetCertificateBySerial(ctx, serialNumber)
	if err != nil {
		return false, err
	}
	if cert == nil {
		return false, nil
	}
	return cert.IsRevoked(), nil
}
