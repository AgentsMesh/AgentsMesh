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

type GenerateReactivationTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"` // seconds
	Command   string `json:"command"`    // Example CLI command
}

func (s *Service) GenerateReactivationToken(ctx context.Context, runnerID, userID int64) (*GenerateReactivationTokenResponse, error) {
	r, err := s.repo.GetByID(ctx, runnerID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get runner for reactivation token", "runner_id", runnerID, "error", err)
		return nil, err
	}
	if r == nil {
		slog.WarnContext(ctx, "runner not found for reactivation token", "runner_id", runnerID)
		return nil, fmt.Errorf("runner not found")
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	expiresAt := time.Now().Add(10 * time.Minute)

	reactivationToken := &runner.ReactivationToken{
		TokenHash: tokenHash,
		RunnerID:  runnerID,
		ExpiresAt: expiresAt,
		CreatedBy: &userID,
	}

	if err := s.repo.CreateReactivationToken(ctx, reactivationToken); err != nil {
		slog.ErrorContext(ctx, "failed to create reactivation token", "runner_id", runnerID, "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to create reactivation token: %w", err)
	}

	slog.InfoContext(ctx, "reactivation token generated", "runner_id", runnerID, "user_id", userID)

	return &GenerateReactivationTokenResponse{
		Token:     token,
		ExpiresIn: 600, // 10 minutes
		Command:   fmt.Sprintf("runner reactivate --token %s", token),
	}, nil
}

type ReactivateRequest struct {
	Token string `json:"token"`
}

type ReactivateResponse struct {
	Certificate   string `json:"certificate"`
	PrivateKey    string `json:"private_key"`
	CACertificate string `json:"ca_certificate"`
}

func (s *Service) Reactivate(ctx context.Context, req *ReactivateRequest, pkiService interfaces.PKICertificateIssuer) (*ReactivateResponse, error) {
	tokenHashBytes := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	reactivationToken, err := s.repo.GetReactivationTokenByHash(ctx, tokenHash)
	if err != nil {
		slog.ErrorContext(ctx, "failed to lookup reactivation token", "error", err)
		return nil, err
	}
	if reactivationToken == nil {
		slog.WarnContext(ctx, "invalid reactivation token presented")
		return nil, ErrInvalidToken
	}

	rowsAffected, err := s.repo.ClaimReactivationToken(ctx, reactivationToken.ID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to claim reactivation token", "token_id", reactivationToken.ID, "error", err)
		return nil, fmt.Errorf("failed to claim token: %w", err)
	}
	if rowsAffected == 0 {
		slog.WarnContext(ctx, "reactivation token expired or already used", "token_id", reactivationToken.ID)
		return nil, ErrTokenExpired
	}

	succeeded := false
	defer func() {
		if !succeeded {
			unclaimCtx, unclaimCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer unclaimCancel()
			_ = s.repo.UnclaimReactivationToken(unclaimCtx, reactivationToken.ID)
		}
	}()

	r, err := s.repo.GetByID(ctx, reactivationToken.RunnerID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get runner for reactivation", "runner_id", reactivationToken.RunnerID, "error", err)
		return nil, err
	}
	if r == nil {
		slog.WarnContext(ctx, "runner not found for reactivation", "runner_id", reactivationToken.RunnerID)
		return nil, fmt.Errorf("runner not found")
	}

	orgSlug, err := s.repo.GetOrgSlug(ctx, r.OrganizationID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get org slug for reactivation", "org_id", r.OrganizationID, "error", err)
		return nil, err
	}
	if orgSlug == "" {
		return nil, fmt.Errorf("organization not found")
	}

	certInfo, err := pkiService.IssueRunnerCertificate(r.NodeID, orgSlug)
	if err != nil {
		slog.ErrorContext(ctx, "failed to issue certificate during reactivation", "runner_id", r.ID, "node_id", r.NodeID, "error", err)
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	cert := &runner.Certificate{
		RunnerID:     r.ID,
		SerialNumber: certInfo.SerialNumber,
		Fingerprint:  certInfo.Fingerprint,
		IssuedAt:     certInfo.IssuedAt,
		ExpiresAt:    certInfo.ExpiresAt,
	}
	if err := s.repo.CreateCertificate(ctx, cert); err != nil {
		slog.ErrorContext(ctx, "failed to save certificate during reactivation", "runner_id", r.ID, "error", err)
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	if err := s.repo.UpdateFields(ctx, r.ID, map[string]interface{}{
		"cert_serial_number": certInfo.SerialNumber,
		"cert_expires_at":    certInfo.ExpiresAt,
	}); err != nil {
		slog.ErrorContext(ctx, "failed to update runner after reactivation", "runner_id", r.ID, "error", err)
		return nil, fmt.Errorf("failed to update runner: %w", err)
	}

	slog.InfoContext(ctx, "runner reactivated successfully", "runner_id", r.ID, "node_id", r.NodeID, "org_slug", orgSlug)

	succeeded = true
	return &ReactivateResponse{
		Certificate:   string(certInfo.CertPEM),
		PrivateKey:    string(certInfo.KeyPEM),
		CACertificate: string(pkiService.CACertPEM()),
	}, nil
}

func (s *Service) CleanupExpiredReactivationTokens(ctx context.Context) error {
	return s.repo.CleanupExpiredReactivationTokens(ctx)
}
