// Package grpc serves Runner connections via gRPC bidi streaming. Handles mTLS directly
// (TLS passthrough) and falls back to metadata for TLS-terminating proxies.
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

type Server struct {
	grpcServer    *grpc.Server
	listener      net.Listener
	logger        *slog.Logger
	config        *config.GRPCConfig
	pkiService    *pki.Service
	runnerAdapter *GRPCRunnerAdapter
}

type ServerDependencies struct {
	Logger             *slog.Logger
	Config             *config.GRPCConfig
	DB                 *gorm.DB
	PKIService         *pki.Service
	RunnerService      RunnerServiceInterface
	OrgService         OrganizationServiceInterface
	AgentsProvider interfaces.AgentsProvider
	ConnManager        *runner.RunnerConnectionManager // 256-shard locks
	MCPDeps            *MCPDependencies                // optional
}

type RunnerServiceInterface interface {
	GetByNodeID(ctx context.Context, nodeID string) (RunnerInfo, error)
	GetByNodeIDAndOrgID(ctx context.Context, nodeID string, orgID int64) (RunnerInfo, error)
	UpdateLastSeen(ctx context.Context, runnerID int64) error
	UpdateAvailableAgents(ctx context.Context, runnerID int64, agents []string) error
	UpdateAgentVersions(ctx context.Context, runnerID int64, versions []runnerDomain.AgentVersion) error
	IsCertificateRevoked(ctx context.Context, serialNumber string) (bool, error)
	UpdateRunnerVersionAndHostInfo(ctx context.Context, runnerID int64, version string, hostInfo map[string]interface{}) error
	MergeAgentVersions(ctx context.Context, runnerID int64, changes map[string]runnerDomain.AgentVersion) error
}

type OrganizationServiceInterface interface {
	GetBySlug(ctx context.Context, slug string) (OrganizationInfo, error)
}

type RunnerInfo struct {
	ID               int64
	NodeID           string
	OrganizationID   int64
	IsEnabled        bool
	CertSerialNumber string
}

type OrganizationInfo struct {
	ID   int64
	Slug string
}

func (s *Server) Stop() {
	s.logger.Info("stopping gRPC server")
	s.grpcServer.GracefulStop()
}

func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}

func (s *Server) RunnerAdapter() *GRPCRunnerAdapter {
	return s.runnerAdapter
}
