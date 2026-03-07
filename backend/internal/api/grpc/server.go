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
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip" // Register gzip compressor/decompressor
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
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
	AgentTypesProvider interfaces.AgentTypesProvider
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


// NewServer creates a new gRPC server for Runner communication.
// The server handles mTLS directly for TLS passthrough mode.
func NewServer(deps *ServerDependencies) (*Server, error) {
	if deps == nil {
		return nil, fmt.Errorf("dependencies are required")
	}
	if deps.Config == nil {
		return nil, fmt.Errorf("gRPC config is required")
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	// Create gRPC server options
	opts := []grpc.ServerOption{
		// Keepalive configuration for long-running streams
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     0,                 // Never close idle connections
			MaxConnectionAge:      0,                 // Never close connections due to age
			MaxConnectionAgeGrace: 0,                 // No grace period
			Time:                  30 * time.Second,  // 30 seconds ping interval
			Timeout:               10 * time.Second,  // 10 seconds ping timeout
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second, // 10 seconds minimum between client pings
			PermitWithoutStream: true,             // Allow pings without active streams
		}),
		// Message size limits
		grpc.MaxRecvMsgSize(16 * 1024 * 1024), // 16MB max receive
		grpc.MaxSendMsgSize(16 * 1024 * 1024), // 16MB max send
		// Interceptors
		grpc.ChainUnaryInterceptor(
			loggingUnaryInterceptor(deps.Logger),
		),
		grpc.ChainStreamInterceptor(
			loggingStreamInterceptor(deps.Logger),
		),
	}

	// Configure mTLS if PKI service is available
	if deps.PKIService != nil {
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{deps.PKIService.ServerCert()},
			ClientCAs:    deps.PKIService.CACertPool(),
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		}
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.Creds(creds))
		deps.Logger.Info("gRPC server configured with mTLS")
	} else {
		deps.Logger.Warn("gRPC server running without TLS (PKI service not available)")
	}

	grpcServer := grpc.NewServer(opts...)

	// Create and register Runner service adapter (delegates to RunnerConnectionManager)
	runnerAdapter := NewGRPCRunnerAdapter(
		deps.Logger,
		deps.DB,
		deps.RunnerService,
		deps.OrgService,
		deps.PKIService,
		deps.AgentTypesProvider,
		deps.ConnManager,
		deps.MCPDeps,
	)

	// Register RunnerService with gRPC server
	runnerAdapter.Register(grpcServer)

	// Note: Init timeout checker is managed by RunnerConnectionManager, not here

	// Enable reflection for debugging/testing
	reflection.Register(grpcServer)

	return &Server{
		grpcServer:    grpcServer,
		logger:        deps.Logger,
		config:        deps.Config,
		pkiService:    deps.PKIService,
		runnerAdapter: runnerAdapter,
	}, nil
}

// Start starts the gRPC server on the configured address.
func (s *Server) Start() error {
	addr := s.config.Address
	if addr == "" {
		addr = ":9090"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	s.logger.Info("gRPC server starting", "address", addr)

	// Serve in goroutine
	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			s.logger.Error("gRPC server error", "error", err)
		}
	}()

	return nil
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
