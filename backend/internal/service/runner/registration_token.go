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

// ==================== Pre-generated Token Registration ====================

// GenerateGRPCRegistrationToken creates a new pre-generated registration token.
func (s *Service) GenerateGRPCRegistrationToken(ctx context.Context, orgID, userID int64, req *GenerateGRPCRegistrationTokenRequest, serverURL string) (*GenerateGRPCRegistrationTokenResponse, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Hash for storage
	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	// Set defaults
	expiresIn := req.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600 // 1 hour default
	}
	maxUses := req.MaxUses
	if maxUses <= 0 {
		maxUses = 1
	}

	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	// Create token record
	regToken := &runner.GRPCRegistrationToken{
		TokenHash:      tokenHash,
		OrganizationID: orgID,
		SingleUse:      req.SingleUse,
		MaxUses:        maxUses,
		ExpiresAt:      expiresAt,
		CreatedBy:      &userID,
	}

	if req.Name != "" {
		regToken.Name = &req.Name
	}
	if len(req.Labels) > 0 {
		regToken.Labels = runner.Labels(req.Labels)
	}

	if err := s.repo.CreateRegistrationToken(ctx, regToken); err != nil {
		slog.Error("failed to create registration token", "org_id", orgID, "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to create registration token: %w", err)
	}

	slog.Info("registration token generated", "org_id", orgID, "user_id", userID)

	return &GenerateGRPCRegistrationTokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Command:   fmt.Sprintf("runner register --server %s --token %s", serverURL, token),
	}, nil
}

// RegisterWithToken registers a new runner using a pre-generated token.
// Uses database transaction with atomic token usage update to prevent race conditions.
func (s *Service) RegisterWithToken(ctx context.Context, req *RegisterWithTokenRequest, pkiService interfaces.PKICertificateIssuer) (*RegisterWithTokenResponse, error) {
	// Hash the provided token
	tokenHashBytes := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	// Find the token first (read-only check)
	regToken, err := s.repo.GetRegistrationTokenByHash(ctx, tokenHash)
	if err != nil {
		slog.Error("failed to lookup registration token", "error", err)
		return nil, err
	}
	if regToken == nil {
		slog.Warn("invalid registration token presented")
		return nil, ErrInvalidToken
	}

	// Basic validation (before transaction)
	if regToken.IsExpired() {
		slog.Warn("expired registration token presented", "token_id", regToken.ID)
		return nil, ErrTokenExpired
	}

	// Get org slug
	orgSlug, err := s.repo.GetOrgSlug(ctx, regToken.OrganizationID)
	if err != nil {
		return nil, err
	}
	if orgSlug == "" {
		return nil, fmt.Errorf("organization not found")
	}

	// Check runner quota
	if s.billingService != nil {
		if err := s.billingService.CheckQuota(ctx, regToken.OrganizationID, "runners", 1); err != nil {
			slog.Warn("runner quota exceeded", "org_id", regToken.OrganizationID)
			return nil, ErrRunnerQuotaExceeded
		}
	}

	// Generate node ID if not provided
	nodeID := req.NodeID
	if nodeID == "" {
		nodeIDBytes := make([]byte, 8)
		if _, err := rand.Read(nodeIDBytes); err != nil {
			return nil, fmt.Errorf("failed to generate node ID: %w", err)
		}
		nodeID = fmt.Sprintf("runner-%s", hex.EncodeToString(nodeIDBytes))
	}

	// Check if runner already exists
	exists, err := s.repo.ExistsByNodeIDAndOrg(ctx, regToken.OrganizationID, nodeID)
	if err != nil {
		return nil, err
	}
	if exists {
		slog.Warn("runner already exists", "org_id", regToken.OrganizationID, "node_id", nodeID)
		return nil, ErrRunnerAlreadyExists
	}

	// Prepare runner and certificate objects
	r := &runner.Runner{
		OrganizationID:     regToken.OrganizationID,
		NodeID:             nodeID,
		Status:             runner.RunnerStatusOffline,
		MaxConcurrentPods:  5,
		Visibility:         runner.VisibilityOrganization,
		RegisteredByUserID: regToken.CreatedBy,
	}

	// Atomic: claim token + create runner.
	// PKI issuance happens inside the callback so that if the token is exhausted,
	// the PKI call is never reached.
	cert := &runner.Certificate{}
	var certPEM, keyPEM []byte
	if err := s.repo.RegisterWithTokenAtomic(ctx, regToken.ID, r, cert, func() error {
		certInfo, err := pkiService.IssueRunnerCertificate(nodeID, orgSlug)
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
	}); err != nil {
		slog.Error("runner registration with token failed", "org_id", regToken.OrganizationID, "node_id", nodeID, "error", err)
		return nil, err
	}

	slog.Info("runner registered with token", "runner_id", r.ID, "org_id", regToken.OrganizationID, "node_id", nodeID, "org_slug", orgSlug)

	return &RegisterWithTokenResponse{
		RunnerID:      r.ID,
		Certificate:   string(certPEM),
		PrivateKey:    string(keyPEM),
		CACertificate: string(pkiService.CACertPEM()),
		OrgSlug:       orgSlug,
	}, nil
}
