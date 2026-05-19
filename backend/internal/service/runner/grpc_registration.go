package runner

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

type RequestAuthURLRequest struct {
	MachineKey string            `json:"machine_key"`
	NodeID     string            `json:"node_id,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type RequestAuthURLResponse struct {
	AuthURL   string `json:"auth_url"`
	AuthKey   string `json:"auth_key"`
	ExpiresIn int    `json:"expires_in"` // seconds
}

type AuthStatusResponse struct {
	Status        string `json:"status"` // "pending", "authorized", "expired"
	NodeID        string `json:"node_id,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"` // ISO 8601 format
	RunnerID      int64  `json:"runner_id,omitempty"`
	Certificate   string `json:"certificate,omitempty"`
	PrivateKey    string `json:"private_key,omitempty"`
	CACertificate string `json:"ca_certificate,omitempty"`
	OrgSlug       string `json:"org_slug,omitempty"`
	GRPCEndpoint  string `json:"grpc_endpoint,omitempty"`
}

type GenerateGRPCRegistrationTokenRequest struct {
	Name      string            `json:"name,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	SingleUse bool              `json:"single_use"`
	MaxUses   int               `json:"max_uses"`
	ExpiresIn int               `json:"expires_in"` // seconds, default 3600 (1 hour)
}

type GenerateGRPCRegistrationTokenResponse struct {
	ID        int64     `json:"id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Command   string    `json:"command"` // Example CLI command
}

type RegisterWithTokenRequest struct {
	Token  string `json:"token"`
	NodeID string `json:"node_id,omitempty"`
}

type RegisterWithTokenResponse struct {
	RunnerID      int64  `json:"runner_id"`
	Certificate   string `json:"certificate"`
	PrivateKey    string `json:"private_key"`
	CACertificate string `json:"ca_certificate"`
	OrgSlug       string `json:"org_slug"`
	GRPCEndpoint  string `json:"grpc_endpoint,omitempty"`
}

func (s *Service) GetRunnerByNodeID(ctx context.Context, nodeID string) (*runner.Runner, error) {
	r, err := s.repo.GetByNodeID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrRunnerNotFound
	}
	return r, nil
}

func (s *Service) ListGRPCRegistrationTokens(ctx context.Context, orgID int64) ([]runner.GRPCRegistrationToken, error) {
	return s.repo.ListRegistrationTokensByOrg(ctx, orgID)
}

func (s *Service) DeleteGRPCRegistrationToken(ctx context.Context, tokenID, orgID int64) error {
	rowsAffected, err := s.repo.DeleteRegistrationToken(ctx, tokenID, orgID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete gRPC registration token", "token_id", tokenID, "org_id", orgID, "error", err)
		return err
	}
	if rowsAffected == 0 {
		return ErrGRPCTokenNotFound
	}
	slog.InfoContext(ctx, "gRPC registration token deleted", "token_id", tokenID, "org_id", orgID)
	return nil
}

func (s *Service) CleanupExpiredPendingAuths(ctx context.Context) error {
	if err := s.repo.CleanupExpiredPendingAuths(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to cleanup expired pending auths", "error", err)
		return err
	}
	return nil
}
