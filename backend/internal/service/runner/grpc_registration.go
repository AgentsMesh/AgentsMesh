package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// ==================== Request/Response Types ====================

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
	NodeID        string `json:"node_id,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"` // ISO 8601 format
	RunnerID      int64  `json:"runner_id,omitempty"`
	Certificate   string `json:"certificate,omitempty"`
	PrivateKey    string `json:"private_key,omitempty"`
	CACertificate string `json:"ca_certificate,omitempty"`
	OrgSlug       string `json:"org_slug,omitempty"`
}

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
// Returns ErrGRPCTokenNotFound if the token doesn't exist.
func (s *Service) DeleteGRPCRegistrationToken(ctx context.Context, tokenID int64) error {
	result := s.db.WithContext(ctx).Delete(&runner.GRPCRegistrationToken{}, tokenID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrGRPCTokenNotFound
	}
	return nil
}

// CleanupExpiredPendingAuths removes expired pending auth records.
func (s *Service) CleanupExpiredPendingAuths(ctx context.Context) error {
	return s.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&runner.PendingAuth{}).Error
}
