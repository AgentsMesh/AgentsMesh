package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
)

func (s *Service) RequestAuthURL(ctx context.Context, req *RequestAuthURLRequest, frontendURL string) (*RequestAuthURLResponse, error) {
	if req.MachineKey == "" {
		return nil, fmt.Errorf("machine_key is required")
	}

	authKeyBytes := make([]byte, 32)
	if _, err := rand.Read(authKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate auth key: %w", err)
	}
	authKey := hex.EncodeToString(authKeyBytes)

	expiresAt := time.Now().Add(15 * time.Minute)

	pendingAuth := &runner.PendingAuth{
		AuthKey:    authKey,
		MachineKey: req.MachineKey,
		ExpiresAt:  expiresAt,
	}

	if req.NodeID != "" {
		pendingAuth.NodeID = &req.NodeID
	}
	if len(req.Labels) > 0 {
		pendingAuth.Labels = runner.Labels(req.Labels)
	}

	if err := s.repo.CreatePendingAuth(ctx, pendingAuth); err != nil {
		return nil, fmt.Errorf("failed to create pending auth: %w", err)
	}

	return &RequestAuthURLResponse{
		AuthURL:   fmt.Sprintf("%s/runners/authorize?key=%s", frontendURL, authKey),
		AuthKey:   authKey,
		ExpiresIn: 900, // 15 minutes in seconds
	}, nil
}

func (s *Service) GetAuthStatus(ctx context.Context, authKey string, pkiService interfaces.PKICertificateIssuer) (*AuthStatusResponse, error) {
	pendingAuth, err := s.repo.GetPendingAuthByKey(ctx, authKey)
	if err != nil {
		return nil, err
	}
	if pendingAuth == nil {
		return nil, ErrAuthRequestNotFound
	}

	if pendingAuth.IsExpired() {
		return &AuthStatusResponse{Status: "expired"}, nil
	}

	if !pendingAuth.Authorized {
		resp := &AuthStatusResponse{
			Status:    "pending",
			ExpiresAt: pendingAuth.ExpiresAt.Format(time.RFC3339),
		}
		if pendingAuth.NodeID != nil {
			resp.NodeID = *pendingAuth.NodeID
		}
		return resp, nil
	}

	if pendingAuth.RunnerID == nil {
		return nil, fmt.Errorf("runner not created yet")
	}

	r, err := s.repo.GetByID(ctx, *pendingAuth.RunnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get runner: %w", err)
	}
	if r == nil {
		return nil, fmt.Errorf("runner not found")
	}

	if r.CertSerialNumber != nil && *r.CertSerialNumber != "" {
		_ = s.repo.RevokeCertificate(ctx, *r.CertSerialNumber, "re-issued: prior poll response lost")
	}

	rowsAffected, err := s.repo.DeleteClaimedPendingAuth(ctx, pendingAuth.ID)
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return &AuthStatusResponse{Status: "pending"}, nil
	}

	var orgSlug string
	if pendingAuth.OrganizationID != nil {
		orgSlug, _ = s.repo.GetOrgSlug(ctx, *pendingAuth.OrganizationID)
	}

	nodeID := r.NodeID
	certInfo, err := pkiService.IssueRunnerCertificate(nodeID, orgSlug)
	if err != nil {
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
		return nil, fmt.Errorf("failed to save certificate: %w", err)
	}

	if err := s.repo.UpdateFields(ctx, r.ID, map[string]interface{}{
		"cert_serial_number": certInfo.SerialNumber,
		"cert_expires_at":    certInfo.ExpiresAt,
	}); err != nil {
		return nil, fmt.Errorf("failed to update runner certificate info: %w", err)
	}

	return &AuthStatusResponse{
		Status:        "authorized",
		RunnerID:      r.ID,
		Certificate:   string(certInfo.CertPEM),
		PrivateKey:    string(certInfo.KeyPEM),
		CACertificate: string(pkiService.CACertPEM()),
		OrgSlug:       orgSlug,
	}, nil
}

func (s *Service) AuthorizeRunner(ctx context.Context, authKey string, orgID int64, userID int64, nodeID string) (*runner.Runner, error) {
	pendingAuth, err := s.repo.GetPendingAuthByKey(ctx, authKey)
	if err != nil {
		return nil, err
	}
	if pendingAuth == nil {
		return nil, ErrAuthRequestNotFound
	}

	if pendingAuth.IsExpired() {
		return nil, ErrAuthRequestExpired
	}

	rowsAffected, err := s.repo.ClaimPendingAuth(ctx, pendingAuth.ID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to claim auth request: %w", err)
	}
	if rowsAffected == 0 {
		return nil, ErrAuthRequestAlreadyAuthorized
	}

	finalNodeID := nodeID
	if finalNodeID == "" && pendingAuth.NodeID != nil {
		finalNodeID = *pendingAuth.NodeID
	}
	if finalNodeID == "" {
		nodeIDBytes := make([]byte, 8)
		if _, err := rand.Read(nodeIDBytes); err != nil {
			return nil, fmt.Errorf("failed to generate node ID: %w", err)
		}
		finalNodeID = fmt.Sprintf("runner-%s", hex.EncodeToString(nodeIDBytes))
	}

	if s.billingService != nil {
		if err := s.billingService.CheckQuota(ctx, orgID, "runners", 1); err != nil {
			return nil, ErrRunnerQuotaExceeded
		}
	}

	exists, err := s.repo.ExistsByNodeIDAndOrg(ctx, orgID, finalNodeID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrRunnerAlreadyExists
	}

	r := &runner.Runner{
		OrganizationID:     orgID,
		NodeID:             finalNodeID,
		Status:             runner.RunnerStatusOffline,
		MaxConcurrentPods:  5,
		Visibility:         runner.VisibilityOrganization,
		RegisteredByUserID: &userID,
	}

	if err := s.repo.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	if err := s.repo.UpdatePendingAuthRunnerID(ctx, pendingAuth.ID, r.ID); err != nil {
		slog.WarnContext(ctx, "Failed to update pending auth runner ID", "error", err)
	}

	return r, nil
}
