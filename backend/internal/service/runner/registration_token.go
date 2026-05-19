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

func (s *Service) GenerateGRPCRegistrationToken(ctx context.Context, orgID, userID int64, req *GenerateGRPCRegistrationTokenRequest, serverURL string) (*GenerateGRPCRegistrationTokenResponse, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	tokenHashBytes := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	expiresIn := req.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 86400
	}
	maxUses := req.MaxUses
	if maxUses <= 0 {
		maxUses = 1
	}

	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

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
		slog.ErrorContext(ctx, "failed to create registration token", "org_id", orgID, "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to create registration token: %w", err)
	}

	slog.InfoContext(ctx, "registration token generated", "org_id", orgID, "user_id", userID)

	return &GenerateGRPCRegistrationTokenResponse{
		ID:        regToken.ID,
		Token:     token,
		ExpiresAt: expiresAt,
		Command:   fmt.Sprintf("runner register --server %s --token %s", serverURL, token),
	}, nil
}

// RegisterWithToken — token claim + runner create + PKI issue must be atomic (token race).
func (s *Service) RegisterWithToken(ctx context.Context, req *RegisterWithTokenRequest, pkiService interfaces.PKICertificateIssuer) (*RegisterWithTokenResponse, error) {
	tokenHashBytes := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	regToken, err := s.repo.GetRegistrationTokenByHash(ctx, tokenHash)
	if err != nil {
		slog.ErrorContext(ctx, "failed to lookup registration token", "error", err)
		return nil, err
	}
	if regToken == nil {
		slog.WarnContext(ctx, "invalid registration token presented")
		return nil, ErrInvalidToken
	}

	if regToken.IsExpired() {
		slog.WarnContext(ctx, "expired registration token presented", "token_id", regToken.ID)
		return nil, ErrTokenExpired
	}

	orgSlug, err := s.repo.GetOrgSlug(ctx, regToken.OrganizationID)
	if err != nil {
		return nil, err
	}
	if orgSlug == "" {
		return nil, fmt.Errorf("organization not found")
	}

	if s.billingService != nil {
		if err := s.billingService.CheckQuota(ctx, regToken.OrganizationID, "runners", 1); err != nil {
			slog.WarnContext(ctx, "runner quota exceeded", "org_id", regToken.OrganizationID)
			return nil, ErrRunnerQuotaExceeded
		}
	}

	nodeID := req.NodeID
	if nodeID == "" {
		nodeIDBytes := make([]byte, 8)
		if _, err := rand.Read(nodeIDBytes); err != nil {
			return nil, fmt.Errorf("failed to generate node ID: %w", err)
		}
		nodeID = fmt.Sprintf("runner-%s", hex.EncodeToString(nodeIDBytes))
	}

	exists, err := s.repo.ExistsByNodeIDAndOrg(ctx, regToken.OrganizationID, nodeID)
	if err != nil {
		return nil, err
	}
	if exists {
		slog.WarnContext(ctx, "runner already exists", "org_id", regToken.OrganizationID, "node_id", nodeID)
		return nil, ErrRunnerAlreadyExists
	}

	r := &runner.Runner{
		OrganizationID:     regToken.OrganizationID,
		NodeID:             nodeID,
		Status:             runner.RunnerStatusOffline,
		MaxConcurrentPods:  5,
		Visibility:         runner.VisibilityOrganization,
		RegisteredByUserID: regToken.CreatedBy,
	}

	// PKI issuance lives inside the atomic claim — exhausted tokens never reach the CA.
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
		slog.ErrorContext(ctx, "runner registration with token failed", "org_id", regToken.OrganizationID, "node_id", nodeID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "runner registered with token", "runner_id", r.ID, "org_id", regToken.OrganizationID, "node_id", nodeID, "org_slug", orgSlug)

	return &RegisterWithTokenResponse{
		RunnerID:      r.ID,
		Certificate:   string(certPEM),
		PrivateKey:    string(keyPEM),
		CACertificate: string(pkiService.CACertPEM()),
		OrgSlug:       orgSlug,
	}, nil
}
