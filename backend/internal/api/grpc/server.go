// Package grpc provides the gRPC server for Runner communication.
// This server handles Runner connections using gRPC bidirectional streaming.
//
// Architecture:
// - Nginx terminates mTLS and passes client certificate info via metadata
// - This server runs without TLS (internal network)
// - Client identity (node_id, org_slug) extracted from gRPC metadata
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/config"
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
}

// RunnerServiceInterface defines the runner service methods needed by gRPC server.
type RunnerServiceInterface interface {
	GetByNodeID(ctx context.Context, nodeID string) (RunnerInfo, error)
	UpdateLastSeen(ctx context.Context, runnerID int64) error
	UpdateAvailableAgents(ctx context.Context, runnerID int64, agents []string) error
	// IsCertificateRevoked checks if a certificate has been revoked.
	// This is called at connection time to enforce certificate revocation.
	IsCertificateRevoked(ctx context.Context, serialNumber string) (bool, error)
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
// The server runs without TLS as Nginx handles mTLS termination.
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
	// Note: No TLS - Nginx terminates mTLS and passes cert info via metadata
	opts := []grpc.ServerOption{
		// Keepalive configuration for long-running streams
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     0,                                    // Never close idle connections
			MaxConnectionAge:      0,                                    // Never close connections due to age
			MaxConnectionAgeGrace: 0,                                    // No grace period
			Time:                  30 * 1000 * 1000 * 1000,              // 30 seconds ping interval
			Timeout:               10 * 1000 * 1000 * 1000,              // 10 seconds ping timeout
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * 1000 * 1000 * 1000, // 10 seconds minimum between client pings
			PermitWithoutStream: true,                   // Allow pings without active streams
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

// ==================== Interceptors ====================

// loggingUnaryInterceptor logs unary RPC calls.
func loggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		logger.Debug("unary RPC call", "method", info.FullMethod)
		return handler(ctx, req)
	}
}

// loggingStreamInterceptor logs stream RPC calls.
func loggingStreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		logger.Debug("stream RPC call", "method", info.FullMethod)
		return handler(srv, ss)
	}
}

// ==================== Metadata Helpers ====================

// MetadataKey constants for gRPC metadata passed by Nginx.
const (
	// MetadataKeyClientCertDN is the client certificate full Subject DN.
	// Set by Nginx from $ssl_client_s_dn, e.g., "CN=dev-runner,O=AgentsMesh,OU=Runner"
	MetadataKeyClientCertDN = "x-client-cert-dn"

	// MetadataKeyClientCertSerial is the client certificate serial number.
	// Set by Nginx from $ssl_client_serial.
	MetadataKeyClientCertSerial = "x-client-cert-serial"

	// MetadataKeyClientCertFingerprint is the client certificate fingerprint.
	// Set by Nginx from $ssl_client_fingerprint.
	MetadataKeyClientCertFingerprint = "x-client-cert-fingerprint"

	// MetadataKeyOrgSlug is the organization slug sent by Runner.
	MetadataKeyOrgSlug = "x-org-slug"

	// MetadataKeyRealIP is the real client IP.
	// Set by Nginx from $remote_addr.
	MetadataKeyRealIP = "x-real-ip"
)

// ClientIdentity holds information extracted from gRPC metadata.
type ClientIdentity struct {
	NodeID           string // From certificate CN
	OrgSlug          string // From Runner metadata
	CertSerialNumber string // From certificate
	CertFingerprint  string // From certificate
	RealIP           string // Client IP
}

// ExtractClientIdentity extracts client identity from gRPC context metadata.
// This information is set by Nginx based on the client certificate.
func ExtractClientIdentity(ctx context.Context) (*ClientIdentity, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no metadata in context")
	}

	identity := &ClientIdentity{}

	// Extract certificate DN and parse CN (node_id) - required
	// Nginx passes $ssl_client_s_dn, e.g., "CN=dev-runner,O=AgentsMesh,OU=Runner"
	if values := md.Get(MetadataKeyClientCertDN); len(values) > 0 && values[0] != "" {
		identity.NodeID = extractCNFromDN(values[0])
	}
	if identity.NodeID == "" {
		return nil, fmt.Errorf("missing client certificate CN (node_id)")
	}

	// Extract org slug - required
	if values := md.Get(MetadataKeyOrgSlug); len(values) > 0 {
		identity.OrgSlug = values[0]
	}
	if identity.OrgSlug == "" {
		return nil, fmt.Errorf("missing org slug")
	}

	// Extract optional fields
	if values := md.Get(MetadataKeyClientCertSerial); len(values) > 0 {
		identity.CertSerialNumber = values[0]
	}
	if values := md.Get(MetadataKeyClientCertFingerprint); len(values) > 0 {
		identity.CertFingerprint = values[0]
	}
	if values := md.Get(MetadataKeyRealIP); len(values) > 0 {
		identity.RealIP = values[0]
	}

	return identity, nil
}

// extractCNFromDN extracts Common Name (CN) from X.509 Subject DN string.
// Supports both formats:
// - OpenSSL default: "/CN=dev-runner/O=AgentsMesh/OU=Runner"
// - RFC 2253: "CN=dev-runner,O=AgentsMesh,OU=Runner"
func extractCNFromDN(dn string) string {
	if dn == "" {
		return ""
	}

	// Try RFC 2253 format first (comma-separated)
	for _, part := range splitDN(dn) {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToUpper(part), "CN=") {
			return strings.TrimPrefix(part, part[:3]) // Handle case variations
		}
	}

	return ""
}

// splitDN splits a DN string by comma or slash separators.
func splitDN(dn string) []string {
	// Check which format is used
	if strings.Contains(dn, "/") && !strings.Contains(dn, ",") {
		// OpenSSL format: "/CN=value/O=value"
		parts := strings.Split(dn, "/")
		// Filter out empty parts
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if p != "" {
				result = append(result, p)
			}
		}
		return result
	}
	// RFC 2253 format: "CN=value,O=value"
	return strings.Split(dn, ",")
}
