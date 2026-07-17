package runner

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

	// Cheap pre-check for a clean error before the runner/org reads; the atomic
	// claim inside ReactivateWithTokenAtomic is the authoritative race-safe guard.
	if reactivationToken.IsExpired() || reactivationToken.IsUsed() {
		slog.WarnContext(ctx, "reactivation token expired or already used", "token_id", reactivationToken.ID)
		return nil, ErrTokenExpired
	}

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

	cert := &runner.Certificate{}
	var certPEM, keyPEM []byte
	err = s.repo.ReactivateWithTokenAtomic(ctx, reactivationToken.ID, r.ID, cert, func() error {
		certInfo, err := pkiService.IssueRunnerCertificate(r.NodeID, orgSlug)
		if err != nil {
			return fmt.Errorf("failed to issue certificate: %w", err)
		}
		cert.SerialNumber = certInfo.SerialNumber
		cert.Fingerprint = certInfo.Fingerprint
		cert.IssuedAt = certInfo.IssuedAt
		cert.ExpiresAt = certInfo.ExpiresAt
		certPEM = certInfo.CertPEM
		keyPEM = certInfo.KeyPEM
		return nil
	})
	if errors.Is(err, runner.ErrReactivationUnavailable) {
		slog.WarnContext(ctx, "reactivation token expired or already used", "token_id", reactivationToken.ID)
		return nil, ErrTokenExpired
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to reactivate runner", "runner_id", r.ID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "runner reactivated successfully", "runner_id", r.ID, "node_id", r.NodeID, "org_slug", orgSlug)

	return &ReactivateResponse{
		Certificate:   string(certPEM),
		PrivateKey:    string(keyPEM),
		CACertificate: string(pkiService.CACertPEM()),
	}, nil
}

func (s *Service) CleanupExpiredReactivationTokens(ctx context.Context) error {
	deleted, err := s.repo.CleanupExpiredReactivationTokens(ctx)
	if err != nil {
		return err
	}
	if deleted > 0 {
		slog.InfoContext(ctx, "purged expired reactivation tokens", "deleted", deleted)
	}
	return nil
}
