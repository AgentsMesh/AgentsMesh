package runner

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
)

// ==================== Tailscale-Style Interactive Registration ====================

// RequestAuthURLRequest represents a request for an authorization URL.
type RequestAuthURLRequest struct {
	MachineKey string            `json:"machine_key"`
	NodeID     string            `json:"node_id,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// RequestAuthURLResponse represents the response with auth URL.
type RequestAuthURLResponse struct {
	AuthURL   string `json:"auth_url"`
	AuthKey   string `json:"auth_key"`
	ExpiresIn int    `json:"expires_in"` // seconds
}

// AuthStatusResponse represents the status of a pending authorization.
type AuthStatusResponse struct {
	Status        string `json:"status"` // "pending", "authorized", "expired"
	RunnerID      int64  `json:"runner_id,omitempty"`
	Certificate   string `json:"certificate,omitempty"`
	PrivateKey    string `json:"private_key,omitempty"`
	CACertificate string `json:"ca_certificate,omitempty"`
	OrgSlug       string `json:"org_slug,omitempty"`
}

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
		return &AuthStatusResponse{Status: "pending"}, nil
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

// ==================== Pre-generated Token Registration ====================

// GenerateGRPCRegistrationTokenRequest represents a request to generate a registration token.
type GenerateGRPCRegistrationTokenRequest struct {
	Name      string            `json:"name,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	SingleUse bool              `json:"single_use"`
	MaxUses   int               `json:"max_uses"`
	ExpiresIn int               `json:"expires_in"` // seconds, default 3600 (1 hour)
}

// GenerateGRPCRegistrationTokenResponse represents the generated token response.
type GenerateGRPCRegistrationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Command   string    `json:"command"` // Example CLI command
}

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

	if err := s.db.WithContext(ctx).Create(regToken).Error; err != nil {
		return nil, fmt.Errorf("failed to create registration token: %w", err)
	}

	return &GenerateGRPCRegistrationTokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Command:   fmt.Sprintf("runner register --server %s --token %s", serverURL, token),
	}, nil
}

// RegisterWithTokenRequest represents a request to register using a pre-generated token.
type RegisterWithTokenRequest struct {
	Token  string `json:"token"`
	NodeID string `json:"node_id,omitempty"`
}

// RegisterWithTokenResponse represents the registration response.
type RegisterWithTokenResponse struct {
	RunnerID      int64  `json:"runner_id"`
	Certificate   string `json:"certificate"`
	PrivateKey    string `json:"private_key"`
	CACertificate string `json:"ca_certificate"`
	OrgSlug       string `json:"org_slug"`
}

// RegisterWithToken registers a new runner using a pre-generated token.
// Uses database transaction with atomic token usage update to prevent race conditions.
func (s *Service) RegisterWithToken(ctx context.Context, req *RegisterWithTokenRequest, pkiService interfaces.PKICertificateIssuer) (*RegisterWithTokenResponse, error) {
	// Hash the provided token
	tokenHashBytes := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	// Find the token first (read-only check)
	var regToken runner.GRPCRegistrationToken
	if err := s.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&regToken).Error; err != nil {
		return nil, ErrInvalidToken
	}

	// Basic validation (before transaction)
	if regToken.IsExpired() {
		return nil, ErrTokenExpired
	}

	// Get org slug
	var org struct {
		ID   int64
		Slug string
	}
	if err := s.db.WithContext(ctx).Table("organizations").
		Select("id, slug").
		Where("id = ?", regToken.OrganizationID).
		First(&org).Error; err != nil {
		return nil, fmt.Errorf("organization not found")
	}

	// Check runner quota
	if s.billingService != nil {
		if err := s.billingService.CheckQuota(ctx, regToken.OrganizationID, "runners", 1); err != nil {
			return nil, ErrRunnerQuotaExceeded
		}
	}

	// Generate node ID if not provided
	nodeID := req.NodeID
	if nodeID == "" {
		nodeIDBytes := make([]byte, 8)
		rand.Read(nodeIDBytes)
		nodeID = fmt.Sprintf("runner-%s", hex.EncodeToString(nodeIDBytes))
	}

	// Check if runner already exists
	var existing runner.Runner
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND node_id = ?", regToken.OrganizationID, nodeID).First(&existing).Error; err == nil {
		return nil, ErrRunnerAlreadyExists
	}

	var result *RegisterWithTokenResponse

	// Use transaction for atomic token usage update and runner creation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Atomic token usage update with condition check
		// This prevents race condition by checking and updating in a single atomic operation
		updateResult := tx.Model(&runner.GRPCRegistrationToken{}).
			Where("id = ? AND (used_count < max_uses OR single_use = false)", regToken.ID).
			Where("expires_at > ?", time.Now()).
			Update("used_count", gorm.Expr("used_count + 1"))

		if updateResult.Error != nil {
			return fmt.Errorf("failed to update token usage: %w", updateResult.Error)
		}

		if updateResult.RowsAffected == 0 {
			// Either token exhausted or expired (race condition case)
			return ErrTokenExhausted
		}

		// Create runner
		r := &runner.Runner{
			OrganizationID:    regToken.OrganizationID,
			NodeID:            nodeID,
			Status:            runner.RunnerStatusOffline,
			MaxConcurrentPods: 5,
		}

		if err := tx.Create(r).Error; err != nil {
			return fmt.Errorf("failed to create runner: %w", err)
		}

		// Issue certificate (outside transaction as it doesn't need DB)
		certInfo, err := pkiService.IssueRunnerCertificate(nodeID, org.Slug)
		if err != nil {
			return fmt.Errorf("failed to issue certificate: %w", err)
		}

		// Save certificate
		cert := &runner.Certificate{
			RunnerID:     r.ID,
			SerialNumber: certInfo.SerialNumber,
			Fingerprint:  certInfo.Fingerprint,
			IssuedAt:     certInfo.IssuedAt,
			ExpiresAt:    certInfo.ExpiresAt,
		}
		if err := tx.Create(cert).Error; err != nil {
			return fmt.Errorf("failed to save certificate: %w", err)
		}

		// Update runner with certificate info
		if err := tx.Model(&runner.Runner{}).
			Where("id = ?", r.ID).
			Updates(map[string]interface{}{
				"cert_serial_number": certInfo.SerialNumber,
				"cert_expires_at":    certInfo.ExpiresAt,
			}).Error; err != nil {
			return fmt.Errorf("failed to update runner certificate info: %w", err)
		}

		result = &RegisterWithTokenResponse{
			RunnerID:      r.ID,
			Certificate:   string(certInfo.CertPEM),
			PrivateKey:    string(certInfo.KeyPEM),
			CACertificate: string(pkiService.CACertPEM()),
			OrgSlug:       org.Slug,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetRunnerByNodeID returns a runner by node_id.
func (s *Service) GetRunnerByNodeID(ctx context.Context, nodeID string) (*runner.Runner, error) {
	var r runner.Runner
	if err := s.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// ListGRPCRegistrationTokens lists all gRPC registration tokens for an organization.
func (s *Service) ListGRPCRegistrationTokens(ctx context.Context, orgID int64) ([]runner.GRPCRegistrationToken, error) {
	var tokens []runner.GRPCRegistrationToken
	if err := s.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

// DeleteGRPCRegistrationToken deletes a gRPC registration token.
func (s *Service) DeleteGRPCRegistrationToken(ctx context.Context, tokenID int64) error {
	return s.db.WithContext(ctx).Delete(&runner.GRPCRegistrationToken{}, tokenID).Error
}

// CleanupExpiredPendingAuths removes expired pending auth records.
func (s *Service) CleanupExpiredPendingAuths(ctx context.Context) error {
	return s.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&runner.PendingAuth{}).Error
}
