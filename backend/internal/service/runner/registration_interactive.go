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
		return nil, fmt.Errorf("auth request not found")
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

	// Delete the pending auth record (one-time use)
	s.db.WithContext(ctx).Delete(&pendingAuth)

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
func (s *Service) AuthorizeRunner(ctx context.Context, authKey string, orgID int64, nodeID string) (*runner.Runner, error) {
	var pendingAuth runner.PendingAuth
	if err := s.db.WithContext(ctx).Where("auth_key = ?", authKey).First(&pendingAuth).Error; err != nil {
		return nil, fmt.Errorf("auth request not found")
	}

	// Check expiration
	if pendingAuth.IsExpired() {
		return nil, fmt.Errorf("auth request expired")
	}

	// Check if already authorized
	if pendingAuth.Authorized {
		return nil, fmt.Errorf("auth request already authorized")
	}

	// Use provided nodeID or generate one
	finalNodeID := nodeID
	if finalNodeID == "" && pendingAuth.NodeID != nil {
		finalNodeID = *pendingAuth.NodeID
	}
	if finalNodeID == "" {
		// Generate a random node ID
		nodeIDBytes := make([]byte, 8)
		rand.Read(nodeIDBytes)
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
		OrganizationID:    orgID,
		NodeID:            finalNodeID,
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		// No auth_token_hash - using mTLS certificates instead
	}

	if err := s.db.WithContext(ctx).Create(r).Error; err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	// Update pending auth
	pendingAuth.Authorized = true
	pendingAuth.OrganizationID = &orgID
	pendingAuth.RunnerID = &r.ID

	if err := s.db.WithContext(ctx).Save(&pendingAuth).Error; err != nil {
		return nil, fmt.Errorf("failed to update pending auth: %w", err)
	}

	return r, nil
}
