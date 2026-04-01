// Package grpc provides the gRPC server for Runner communication.
// This server handles Runner connections using gRPC bidirectional streaming.
//
// Architecture:
// - Server handles mTLS directly (TLS passthrough from reverse proxy)
// - Client identity extracted from TLS peer certificate
// - Supports both modes: direct mTLS or metadata-based (when behind TLS-terminating proxy)
package grpc

import (
	"context"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// Server wraps the gRPC server with Runner-specific configuration.
type Server struct {
	grpcServer    *grpc.Server
	listener      net.Listener
	logger        *slog.Logger
	config        *config.GRPCConfig
	pkiService    *pki.Service
	runnerAdapter *GRPCRunnerAdapter
}

// ServerDependencies holds dependencies for creating the gRPC server.
type ServerDependencies struct {
	Logger             *slog.Logger
	Config             *config.GRPCConfig
	DB                 *gorm.DB // Database connection for audit logging
	PKIService         *pki.Service
	RunnerService      RunnerServiceInterface
	OrgService         OrganizationServiceInterface
	AgentsProvider interfaces.AgentsProvider
	ConnManager        *runner.RunnerConnectionManager // Connection manager with 256-shard locks
	MCPDeps            *MCPDependencies                // Optional MCP service dependencies
}

// RunnerServiceInterface defines the runner service methods needed by gRPC server.
type RunnerServiceInterface interface {
	GetByNodeID(ctx context.Context, nodeID string) (RunnerInfo, error)
	GetByNodeIDAndOrgID(ctx context.Context, nodeID string, orgID int64) (RunnerInfo, error)
	UpdateLastSeen(ctx context.Context, runnerID int64) error
	UpdateAvailableAgents(ctx context.Context, runnerID int64, agents []string) error
	UpdateAgentVersions(ctx context.Context, runnerID int64, versions []runnerDomain.AgentVersion) error
	// IsCertificateRevoked checks if a certificate has been revoked.
	// This is called at connection time to enforce certificate revocation.
	IsCertificateRevoked(ctx context.Context, serialNumber string) (bool, error)
	// UpdateRunnerVersionAndHostInfo persists runner version and host info from the gRPC handshake.
	UpdateRunnerVersionAndHostInfo(ctx context.Context, runnerID int64, version string, hostInfo map[string]interface{}) error
	// MergeAgentVersions merges delta agent version updates into existing versions.
	// Entries where both Version and Path are empty are treated as removals.
	MergeAgentVersions(ctx context.Context, runnerID int64, changes map[string]runnerDomain.AgentVersion) error
}

// OrganizationServiceInterface defines the organization service methods needed.
type OrganizationServiceInterface interface {
	GetBySlug(ctx context.Context, slug string) (OrganizationInfo, error)
}

// RunnerInfo contains Runner information returned by the service.
type RunnerInfo struct {
	ID               int64
	NodeID           string
	OrganizationID   int64
	IsEnabled        bool
	CertSerialNumber string
}

// OrganizationInfo contains Organization information.
type OrganizationInfo struct {
	ID   int64
	Slug string
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.logger.Info("stopping gRPC server")
	// Note: Init timeout checker is managed by RunnerConnectionManager
	s.grpcServer.GracefulStop()
}

// GRPCServer returns the underlying gRPC server for registration.
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}

// RunnerAdapter returns the gRPC Runner adapter.
func (s *Server) RunnerAdapter() *GRPCRunnerAdapter {
	return s.runnerAdapter
}
