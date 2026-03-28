package runner

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
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
	r, err := s.repo.GetByID(ctx, runnerID)
	if err != nil {
		slog.Error("failed to get runner for reactivation token", "runner_id", runnerID, "error", err)
		return nil, err
	}
	if r == nil {
		slog.Warn("runner not found for reactivation token", "runner_id", runnerID)
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

	if err := s.repo.CreateReactivationToken(ctx, reactivationToken); err != nil {
		slog.Error("failed to create reactivation token", "runner_id", runnerID, "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to create reactivation token: %w", err)
	}

	slog.Info("reactivation token generated", "runner_id", runnerID, "user_id", userID)

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
	reactivationToken, err := s.repo.GetReactivationTokenByHash(ctx, tokenHash)
	if err != nil {
		slog.Error("failed to lookup reactivation token", "error", err)
		return nil, err
	}
	if reactivationToken == nil {
		slog.Warn("invalid reactivation token presented")
		return nil, ErrInvalidToken
	}

	// Atomic claim: mark token as used only if it hasn't been used yet and isn't expired.
	rowsAffected, err := s.repo.ClaimReactivationToken(ctx, reactivationToken.ID)
	if err != nil {
		slog.Error("failed to claim reactivation token", "token_id", reactivationToken.ID, "error", err)
		return nil, fmt.Errorf("failed to claim token: %w", err)
	}
	if rowsAffected == 0 {
		slog.Warn("reactivation token expired or already used", "token_id", reactivationToken.ID)
		return nil, ErrTokenExpired
	}

	// Compensating action: if any subsequent step fails, unclaim the token
	succeeded := false
	defer func() {
		if !succeeded {
			unclaimCtx, unclaimCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer unclaimCancel()
			_ = s.repo.UnclaimReactivationToken(unclaimCtx, reactivationToken.ID)
		}
	}()

	// Get runner
	r, err := s.repo.GetByID(ctx, reactivationToken.RunnerID)
	if err != nil {
		slog.Error("failed to get runner for reactivation", "runner_id", reactivationToken.RunnerID, "error", err)
		return nil, err
	}
	if r == nil {
		slog.Warn("runner not found for reactivation", "runner_id", reactivationToken.RunnerID)
		return nil, fmt.Errorf("runner not found")
	}

	// Get org slug
	orgSlug, err := s.repo.GetOrgSlug(ctx, r.OrganizationID)
	if err != nil {
		slog.Error("failed to get org slug for reactivation", "org_id", r.OrganizationID, "error", err)
		return nil, err
	}
	if orgSlug == "" {
		return nil, fmt.Errorf("organization not found")
	}

	// Issue new certificate
	certInfo, err := pkiService.IssueRunnerCertificate(r.NodeID, orgSlug)
	if err != nil {
		slog.Error("failed to issue certificate during reactivation", "runner_id", r.ID, "node_id", r.NodeID, "error", err)
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
	if err := s.repo.CreateCertificate(ctx, cert); err != nil {
		slog.Error("failed to save certificate during reactivation", "runner_id", r.ID, "error", err)
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	// Update runner
	if err := s.repo.UpdateFields(ctx, r.ID, map[string]interface{}{
		"cert_serial_number": certInfo.SerialNumber,
		"cert_expires_at":    certInfo.ExpiresAt,
	}); err != nil {
		slog.Error("failed to update runner after reactivation", "runner_id", r.ID, "error", err)
		return nil, fmt.Errorf("failed to update runner: %w", err)
	}

	slog.Info("runner reactivated successfully", "runner_id", r.ID, "node_id", r.NodeID, "org_slug", orgSlug)

	succeeded = true
	return &ReactivateResponse{
		Certificate:   string(certInfo.CertPEM),
		PrivateKey:    string(certInfo.KeyPEM),
		CACertificate: string(pkiService.CACertPEM()),
	}, nil
}

// CleanupExpiredReactivationTokens removes expired reactivation tokens.
func (s *Service) CleanupExpiredReactivationTokens(ctx context.Context) error {
	return s.repo.CleanupExpiredReactivationTokens(ctx)
}
