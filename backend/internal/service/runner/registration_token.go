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

	if err := s.db.WithContext(ctx).Create(regToken).Error; err != nil {
		return nil, fmt.Errorf("failed to create registration token: %w", err)
	}

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
		if _, err := rand.Read(nodeIDBytes); err != nil {
			return nil, fmt.Errorf("failed to generate node ID: %w", err)
		}
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
		// This prevents race condition by checking and updating in a single atomic operation.
		// Only check used_count < max_uses; single_use tokens already have max_uses = 1.
		updateResult := tx.Model(&runner.GRPCRegistrationToken{}).
			Where("id = ? AND used_count < max_uses", regToken.ID).
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
			OrganizationID:     regToken.OrganizationID,
			NodeID:             nodeID,
			Status:             runner.RunnerStatusOffline,
			MaxConcurrentPods:  5,
			Visibility:         runner.VisibilityOrganization,
			RegisteredByUserID: regToken.CreatedBy,
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
