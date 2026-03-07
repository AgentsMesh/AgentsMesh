package runner

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
)

// ==================== Reactivation (Expired Certificate Recovery) ====================

// GenerateReactivationTokenResponse represents the reactivation token response.
type GenerateReactivationTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"` // seconds
	Command   string `json:"command"`    // Example CLI command
}

// GenerateReactivationToken creates a one-time token for reactivating a runner with expired certificate.
func (s *Service) GenerateReactivationToken(ctx context.Context, runnerID, userID int64) (*GenerateReactivationTokenResponse, error) {
	// Verify runner exists
	var r runner.Runner
	if err := s.db.WithContext(ctx).First(&r, runnerID).Error; err != nil {
		return nil, fmt.Errorf("runner not found")
	}

	// Generate token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Hash for storage
	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	// 10 minutes expiration
	expiresAt := time.Now().Add(10 * time.Minute)

	// Create reactivation token
	reactivationToken := &runner.ReactivationToken{
		TokenHash: tokenHash,
		RunnerID:  runnerID,
		ExpiresAt: expiresAt,
		CreatedBy: &userID,
	}

	if err := s.db.WithContext(ctx).Create(reactivationToken).Error; err != nil {
		return nil, fmt.Errorf("failed to create reactivation token: %w", err)
	}

	return &GenerateReactivationTokenResponse{
		Token:     token,
		ExpiresIn: 600, // 10 minutes
		Command:   fmt.Sprintf("runner reactivate --token %s", token),
	}, nil
}

// ReactivateRequest represents a request to reactivate a runner.
type ReactivateRequest struct {
	Token string `json:"token"`
}

// ReactivateResponse represents the reactivation response.
type ReactivateResponse struct {
	Certificate   string `json:"certificate"`
	PrivateKey    string `json:"private_key"`
	CACertificate string `json:"ca_certificate"`
}

// Reactivate reactivates a runner using a one-time token.
func (s *Service) Reactivate(ctx context.Context, req *ReactivateRequest, pkiService interfaces.PKICertificateIssuer) (*ReactivateResponse, error) {
	// Hash the provided token
	tokenHashBytes := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	// Find the token
	var reactivationToken runner.ReactivationToken
	if err := s.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&reactivationToken).Error; err != nil {
		return nil, ErrInvalidToken
	}

	// Atomic claim: mark token as used only if it hasn't been used yet and isn't expired.
	// This prevents TOCTOU race where two concurrent requests both pass IsValid() check.
	now := time.Now()
	claimResult := s.db.WithContext(ctx).Model(&reactivationToken).
		Where("used_at IS NULL AND expires_at > ?", now).
		Update("used_at", now)
	if claimResult.Error != nil {
		return nil, fmt.Errorf("failed to claim token: %w", claimResult.Error)
	}
	if claimResult.RowsAffected == 0 {
		return nil, ErrTokenExpired
	}

	// Compensating action: if any subsequent step fails, unclaim the token
	// so the user can retry. Without this, a transient PKI/DB failure would
	// permanently consume the one-time token with no certificate issued.
	succeeded := false
	defer func() {
		if !succeeded {
			unclaimCtx, unclaimCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer unclaimCancel()
			_ = s.db.WithContext(unclaimCtx).
				Model(&reactivationToken).
				Update("used_at", nil).Error
		}
	}()

	// Get runner
	var r runner.Runner
	if err := s.db.WithContext(ctx).First(&r, reactivationToken.RunnerID).Error; err != nil {
		return nil, fmt.Errorf("runner not found")
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
	certInfo, err := pkiService.IssueRunnerCertificate(r.NodeID, org.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Save certificate
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

	succeeded = true
	return &ReactivateResponse{
		Certificate:   string(certInfo.CertPEM),
		PrivateKey:    string(certInfo.KeyPEM),
		CACertificate: string(pkiService.CACertPEM()),
	}, nil
}

// CleanupExpiredReactivationTokens removes expired reactivation tokens.
func (s *Service) CleanupExpiredReactivationTokens(ctx context.Context) error {
	return s.db.WithContext(ctx).
		Where("expires_at < ? OR used_at IS NOT NULL", time.Now()).
		Delete(&runner.ReactivationToken{}).Error
}
