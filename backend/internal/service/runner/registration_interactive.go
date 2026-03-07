package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
)

// ==================== Tailscale-Style Interactive Registration ====================

// RequestAuthURL creates a pending auth request and returns an authorization URL.
// This is step 1 of Tailscale-style interactive registration.
func (s *Service) RequestAuthURL(ctx context.Context, req *RequestAuthURLRequest, frontendURL string) (*RequestAuthURLResponse, error) {
	if req.MachineKey == "" {
		return nil, fmt.Errorf("machine_key is required")
	}

	// Generate unique auth key
	authKeyBytes := make([]byte, 32)
	if _, err := rand.Read(authKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate auth key: %w", err)
	}
	authKey := hex.EncodeToString(authKeyBytes)

	// Set expiration (15 minutes)
	expiresAt := time.Now().Add(15 * time.Minute)

	// Create pending auth record
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

	if err := s.db.WithContext(ctx).Create(pendingAuth).Error; err != nil {
		return nil, fmt.Errorf("failed to create pending auth: %w", err)
	}

	return &RequestAuthURLResponse{
		AuthURL:   fmt.Sprintf("%s/runners/authorize?key=%s", frontendURL, authKey),
		AuthKey:   authKey,
		ExpiresIn: 900, // 15 minutes in seconds
	}, nil
}

// GetAuthStatus returns the current status of a pending authorization.
// This is called by Runner polling for authorization completion.
func (s *Service) GetAuthStatus(ctx context.Context, authKey string, pkiService interfaces.PKICertificateIssuer) (*AuthStatusResponse, error) {
	var pendingAuth runner.PendingAuth
	if err := s.db.WithContext(ctx).Where("auth_key = ?", authKey).First(&pendingAuth).Error; err != nil {
		return nil, ErrAuthRequestNotFound
	}

	// Check expiration
	if pendingAuth.IsExpired() {
		return &AuthStatusResponse{Status: "expired"}, nil
	}

	// Check if authorized
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

	// Get the created runner
	if pendingAuth.RunnerID == nil {
		return nil, fmt.Errorf("runner not created yet")
	}

	var r runner.Runner
	if err := s.db.WithContext(ctx).First(&r, *pendingAuth.RunnerID).Error; err != nil {
		return nil, fmt.Errorf("failed to get runner: %w", err)
	}

	// Handle lost HTTP response: if the runner already has a certificate from
	// a prior successful poll whose response was lost, revoke the orphaned cert
	// and re-issue. We cannot return the original (private key is not stored),
	// so re-issuing is the only way to recover without deadlocking the runner.
	if r.CertSerialNumber != nil && *r.CertSerialNumber != "" {
		now := time.Now()
		_ = s.db.WithContext(ctx).Model(&runner.Certificate{}).
			Where("serial_number = ?", *r.CertSerialNumber).
			Updates(map[string]interface{}{
				"revoked_at":        now,
				"revocation_reason": "re-issued: prior poll response lost",
			}).Error // best-effort revocation; proceed regardless
	}

	// Atomic claim: delete pendingAuth before cert issuance to prevent concurrent
	// polls from each issuing a certificate. Only one request can succeed.
	deleteResult := s.db.WithContext(ctx).
		Where("id = ? AND authorized = true", pendingAuth.ID).
		Delete(&runner.PendingAuth{})
	if deleteResult.RowsAffected == 0 {
		// Concurrent poll already consumed this auth record; return pending
		// so the runner's HTTP client retries with the other response.
		return &AuthStatusResponse{Status: "pending"}, nil
	}

	// Get org slug
	var orgSlug string
	if pendingAuth.OrganizationID != nil {
		var org struct {
			Slug string
		}
		if err := s.db.WithContext(ctx).Table("organizations").
			Select("slug").
			Where("id = ?", *pendingAuth.OrganizationID).
			First(&org).Error; err == nil {
			orgSlug = org.Slug
		}
	}

	// Issue certificate
	nodeID := r.NodeID
	certInfo, err := pkiService.IssueRunnerCertificate(nodeID, orgSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Save certificate to database
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

	// Update runner with certificate info
	if err := s.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ?", r.ID).
		Updates(map[string]interface{}{
			"cert_serial_number": certInfo.SerialNumber,
			"cert_expires_at":    certInfo.ExpiresAt,
		}).Error; err != nil {
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

// AuthorizeRunner authorizes a pending auth request (called from Web UI).
// This is step 2 of Tailscale-style interactive registration.
// userID is the ID of the user performing the authorization, recorded as RegisteredByUserID.
func (s *Service) AuthorizeRunner(ctx context.Context, authKey string, orgID int64, userID int64, nodeID string) (*runner.Runner, error) {
	var pendingAuth runner.PendingAuth
	if err := s.db.WithContext(ctx).Where("auth_key = ?", authKey).First(&pendingAuth).Error; err != nil {
		return nil, ErrAuthRequestNotFound
	}

	// Check expiration (informational — the atomic claim below also checks)
	if pendingAuth.IsExpired() {
		return nil, ErrAuthRequestExpired
	}

	// Atomic claim: set authorized=true only if currently false and not expired.
	// Prevents TOCTOU race from concurrent Web UI double-clicks.
	claimResult := s.db.WithContext(ctx).Model(&runner.PendingAuth{}).
		Where("id = ? AND authorized = false AND expires_at > ?", pendingAuth.ID, time.Now()).
		Updates(map[string]interface{}{
			"authorized":      true,
			"organization_id": orgID,
		})
	if claimResult.Error != nil {
		return nil, fmt.Errorf("failed to claim auth request: %w", claimResult.Error)
	}
	if claimResult.RowsAffected == 0 {
		return nil, ErrAuthRequestAlreadyAuthorized
	}

	// Use provided nodeID or generate one
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

	// Check runner quota
	if s.billingService != nil {
		if err := s.billingService.CheckQuota(ctx, orgID, "runners", 1); err != nil {
			return nil, ErrRunnerQuotaExceeded
		}
	}

	// Check if runner already exists
	var existing runner.Runner
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND node_id = ?", orgID, finalNodeID).First(&existing).Error; err == nil {
		return nil, ErrRunnerAlreadyExists
	}

	// Create the runner
	r := &runner.Runner{
		OrganizationID:     orgID,
		NodeID:             finalNodeID,
		Status:             runner.RunnerStatusOffline,
		MaxConcurrentPods:  5,
		Visibility:         runner.VisibilityOrganization,
		RegisteredByUserID: &userID,
	}

	if err := s.db.WithContext(ctx).Create(r).Error; err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	// Update pending auth with runner ID
	if err := s.db.WithContext(ctx).Model(&runner.PendingAuth{}).
		Where("id = ?", pendingAuth.ID).
		Update("runner_id", r.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to update pending auth: %w", err)
	}

	return r, nil
}
